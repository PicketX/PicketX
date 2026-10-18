// SPDX-License-Identifier: Apache-2.0
package agentsync

import (
	"bytes"
	"context"
	"io"
	"time"

	agentv1 "github.com/PicketX/PicketX/pkg/protocol/agent/v1alpha1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

type session struct {
	key        [32]byte
	credential Credential
	updates    chan struct{}
	terminal   chan struct{}
	reason     codes.Code // guarded by Service.mu; terminal close publishes it
}

func (ss *session) wake() {
	select {
	case ss.updates <- struct{}{}:
	default:
	}
}
func (ss *session) end(reason codes.Code) {
	select {
	case <-ss.terminal:
		return
	default:
		ss.reason = reason
		close(ss.terminal)
	}
}

type incoming struct {
	frame *agentv1.AgentFrame
	err   error
}

func (s *Service) Connect(stream grpc.BidiStreamingServer[agentv1.AgentFrame, agentv1.ControllerFrame]) error {
	md, _ := metadata.FromIncomingContext(stream.Context())
	ss, e := s.admit(md)
	if e != nil {
		return e
	}
	defer func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.sessions[ss.credential.AgentRef] == ss {
			delete(s.sessions, ss.credential.AgentRef)
		}
	}()
	// Returning the RPC cancels gRPC's transport context, interrupting even blocked
	// Send/Recv calls. Merely cancelling a child context would not interrupt Send.
	ctx, cancel := context.WithCancel(stream.Context())
	defer cancel()
	received := make(chan incoming)
	go func() {
		for {
			f, e := stream.Recv()
			select {
			case received <- incoming{f, e}:
			case <-ctx.Done():
				return
			}
			if e != nil {
				return
			}
		}
	}()
	helloTimer := time.NewTimer(s.limits.HelloTimeout)
	defer helloTimer.Stop()
	var messages window
	select {
	case <-ss.terminal:
		return terminalError(ss)
	case <-ctx.Done():
		return status.FromContextError(ctx.Err()).Err()
	case <-helloTimer.C:
		return status.Error(codes.DeadlineExceeded, "Agent Hello timeout")
	case v := <-received:
		if v.err != nil {
			return receiveError(v.err)
		}
		if v.frame.GetHello() == nil || v.frame.GetHello().ProtocolVersion != 1 {
			return invalidMessage()
		}
		messages.take(time.Now(), s.limits.MessagesPerSecond)
	}
	outgoing := make(chan *agentv1.Snapshot)
	sent := make(chan error)
	go func() {
		for {
			select {
			case snapshot := <-outgoing:
				e := stream.Send(&agentv1.ControllerFrame{Snapshot: snapshot})
				select {
				case sent <- e:
				case <-ctx.Done():
					return
				}
				if e != nil {
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	timer := time.NewTimer(s.limits.SendTimeout)
	if !timer.Stop() {
		<-timer.C
	}
	defer timer.Stop()
	var timeout <-chan time.Time
	var last, inflight, next *agentv1.Snapshot
	pending := true
	for {
		var send chan *agentv1.Snapshot
		if pending && inflight == nil {
			s.mu.Lock()
			next = s.snapshots[ss.credential.AgentRef]
			active := s.sessions[ss.credential.AgentRef] == ss
			s.mu.Unlock()
			if !active {
				return terminalError(ss)
			}
			if next != nil && (last == nil || last.Revision != next.Revision) {
				send = outgoing
			} else {
				pending = false
			}
		}
		// A report may arrive just before Send returns. Wait for send completion so
		// correlation never rejects the first valid report because of scheduling.
		var recv <-chan incoming = received
		if inflight != nil {
			recv = nil
		}
		select {
		case <-ss.terminal:
			return terminalError(ss)
		case <-ctx.Done():
			return status.FromContextError(ctx.Err()).Err()
		case <-timeout:
			return status.Error(codes.DeadlineExceeded, "Agent send timeout")
		case <-ss.updates:
			pending = true
		case send <- next:
			inflight = next
			pending = false
			timer.Reset(s.limits.SendTimeout)
			timeout = timer.C
		case e := <-sent:
			if e != nil {
				return e
			}
			last = inflight
			inflight = nil
			timeout = nil
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
		case v := <-recv:
			if v.err != nil {
				return receiveError(v.err)
			}
			if !messages.take(time.Now(), s.limits.MessagesPerSecond) {
				return status.Error(codes.ResourceExhausted, "Agent message limit")
			}
			r := v.frame.GetReport()
			if last == nil || r == nil || r.Revision != last.Revision || !bytes.Equal(r.Sha256, last.Sha256) || r.Phase < agentv1.Report_RECEIVED || r.Phase > agentv1.Report_FAILED {
				return invalidMessage()
			}
			observedAt, e := time.Parse(time.RFC3339Nano, r.ObservedAt)
			if e != nil || observedAt.IsZero() {
				return invalidMessage()
			}
			s.mu.Lock()
			if s.closed || s.sessions[ss.credential.AgentRef] != ss {
				s.mu.Unlock()
				return terminalError(ss)
			}
			s.reports[ss.credential.AgentRef] = observed{report: proto.Clone(r).(*agentv1.Report), received: time.Now().UTC(), session: ss}
			s.mu.Unlock()
		}
	}
}
func invalidMessage() error { return status.Error(codes.InvalidArgument, "invalid Agent message") }
func receiveError(e error) error {
	if e == io.EOF {
		return nil
	}
	return e
}
func terminalError(ss *session) error {
	<-ss.terminal
	return status.Error(ss.reason, "Agent stream ended")
}
