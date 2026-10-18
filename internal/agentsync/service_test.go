// SPDX-License-Identifier: Apache-2.0
package agentsync

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"google.golang.org/grpc/stats"
	"math/big"
	"net"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/PicketX/PicketX/pkg/configuration"
	agentv1 "github.com/PicketX/PicketX/pkg/protocol/agent/v1alpha1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

const tokenA = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
const tokenB = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
const authority = "static:example-v1"

func limits() Limits {
	return Limits{MaxCredentials: 8, MaxStreams: 4, MaxSnapshotBytes: 4 << 20, EstablishPerSecond: 100, MessagesPerSecond: 100, HelloTimeout: time.Second, SendTimeout: time.Second}
}
func credential(id, agent, token string) Credential {
	return Credential{ID: id, AgentRef: agent, SHA256: sha256.Sum256([]byte(token))}
}
func candidate(t *testing.T, empty bool) *configuration.Candidate {
	t.Helper()
	file := "gitea.yaml"
	if empty {
		file = "empty.yaml"
	}
	data, e := os.ReadFile("../../examples/static/" + file)
	if e != nil {
		t.Fatal(e)
	}
	raw, e := os.ReadFile("../../examples/static/facts.json")
	if e != nil {
		t.Fatal(e)
	}
	var f configuration.Facts
	if e = json.Unmarshal(raw, &f); e != nil {
		t.Fatal(e)
	}
	b, e := configuration.Decode(data)
	if e != nil {
		t.Fatal(e)
	}
	c, e := configuration.Compile(b, f)
	if e != nil {
		t.Fatal(e)
	}
	return c
}
func setup(t *testing.T, l Limits) (*Service, agentv1.AgentSyncClient) {
	t.Helper()
	s, e := New(authority, l)
	if e != nil {
		t.Fatal(e)
	}
	if e = s.ReplaceCredentials([]Credential{credential("one", "agent-a", tokenA)}); e != nil {
		t.Fatal(e)
	}
	t.Cleanup(s.Close)
	return s, client(t, s, nil, nil)
}
func client(t *testing.T, s *Service, serverTLS, clientTLS *tls.Config, extra ...grpc.ServerOption) agentv1.AgentSyncClient {
	t.Helper()
	listener := bufconn.Listen(1 << 20)
	opts := []grpc.ServerOption{grpc.MaxRecvMsgSize(4 << 20), grpc.MaxSendMsgSize(4 << 20)}
	if serverTLS != nil {
		opts = append(opts, grpc.Creds(credentials.NewTLS(serverTLS)))
	}
	opts = append(opts, extra...)
	server := grpc.NewServer(opts...)
	agentv1.RegisterAgentSyncServer(server, s)
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() { server.Stop(); _ = listener.Close() })
	transport := insecure.NewCredentials()
	if clientTLS != nil {
		transport = credentials.NewTLS(clientTLS)
	}
	cc, e := grpc.NewClient("passthrough:///agent", grpc.WithTransportCredentials(transport), grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }), grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(4<<20), grpc.MaxCallSendMsgSize(4<<20)))
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = cc.Close() })
	return agentv1.NewAgentSyncClient(cc)
}
func connect(t *testing.T, c agentv1.AgentSyncClient, token string, hello bool) grpc.BidiStreamingClient[agentv1.AgentFrame, agentv1.ControllerFrame] {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	t.Cleanup(cancel)
	if token != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+token)
	}
	st, e := c.Connect(ctx)
	if e != nil {
		t.Fatal(e)
	}
	if hello {
		if e = st.Send(&agentv1.AgentFrame{Body: &agentv1.AgentFrame_Hello{Hello: &agentv1.Hello{ProtocolVersion: 1}}}); e != nil {
			t.Fatal(e)
		}
	}
	return st
}
func recv(t *testing.T, st grpc.BidiStreamingClient[agentv1.AgentFrame, agentv1.ControllerFrame]) *agentv1.Snapshot {
	t.Helper()
	f, e := st.Recv()
	if e != nil || f.GetSnapshot() == nil {
		t.Fatal(f, e)
	}
	return f.Snapshot
}
func code(t *testing.T, st grpc.BidiStreamingClient[agentv1.AgentFrame, agentv1.ControllerFrame], want codes.Code) {
	t.Helper()
	_, e := st.Recv()
	if status.Code(e) != want {
		t.Fatalf("got %v want %v", e, want)
	}
}
func sendReport(t *testing.T, st grpc.BidiStreamingClient[agentv1.AgentFrame, agentv1.ControllerFrame], s *agentv1.Snapshot) {
	t.Helper()
	if e := st.Send(&agentv1.AgentFrame{Body: &agentv1.AgentFrame_Report{Report: &agentv1.Report{Revision: s.Revision, Sha256: s.Sha256, Phase: agentv1.Report_APPLIED, ObservedAt: time.Now().UTC().Format(time.RFC3339Nano)}}}); e != nil {
		t.Fatal(e)
	}
}
func waitReport(t *testing.T, s *Service, agent string) Observation {
	t.Helper()
	until := time.Now().Add(time.Second)
	for time.Now().Before(until) {
		if o, ok := s.Report(agent); ok {
			return o
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("report not observed")
	return Observation{}
}
func TestStreamScopedSnapshotsReportsReconnectAndWithdrawal(t *testing.T) {
	s, c := setup(t, limits())
	if e := s.Publish(candidate(t, false), 1); e != nil {
		t.Fatal(e)
	}
	a := connect(t, c, tokenA, true)
	first := recv(t, a)
	if first.AgentRef != "agent-a" || len(first.Targets) != 2 {
		t.Fatal(first)
	}
	sendReport(t, a, first)
	o := waitReport(t, s, "agent-a")
	if !o.Connected || o.ReceivedAt.IsZero() || o.Report.Phase != agentv1.Report_APPLIED {
		t.Fatal(o)
	}
	o.Report.Sha256[0] ^= 0xff
	again, _ := s.Report("agent-a")
	if !bytes.Equal(again.Report.Sha256, first.Sha256) {
		t.Fatal("report alias")
	}
	if e := s.ReplaceCredentials([]Credential{credential("one", "agent-a", tokenA), credential("two", "agent-b", tokenB)}); e != nil {
		t.Fatal(e)
	}
	b := connect(t, c, tokenB, true)
	if x := recv(t, b); x.AgentRef != "agent-b" || len(x.Targets) != 0 {
		t.Fatal(x)
	}
	replacement := connect(t, c, tokenA, true)
	if x := recv(t, replacement); x.Revision != 1 || !bytes.Equal(x.Sha256, first.Sha256) {
		t.Fatal(x)
	}
	code(t, a, codes.Aborted)
	stale, _ := s.Report("agent-a")
	if stale.Connected {
		t.Fatal("old report represented as current stream")
	}
	if e := s.Publish(candidate(t, true), 2); e != nil {
		t.Fatal(e)
	}
	if x := recv(t, replacement); x.Revision != 2 || len(x.Targets) != 0 {
		t.Fatal(x)
	}
	reconnect := connect(t, c, tokenA, true)
	if x := recv(t, reconnect); x.Revision != 2 || len(x.Targets) != 0 {
		t.Fatal(x)
	}
}
func TestRejectAuthenticationAndProtocol(t *testing.T) {
	s, c := setup(t, limits())
	_ = s.Publish(candidate(t, false), 1)
	for _, token := range []string{"", strings.Repeat("z", 32), "short", tokenA + " ", strings.Repeat("a", 257)} {
		st := connect(t, c, token, false)
		code(t, st, codes.Unauthenticated)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+tokenA, "authorization", "Bearer "+tokenA)
	st, e := c.Connect(ctx)
	if e != nil {
		t.Fatal(e)
	}
	code(t, st, codes.Unauthenticated)
	for _, f := range []*agentv1.AgentFrame{{}, {Body: &agentv1.AgentFrame_Hello{Hello: &agentv1.Hello{ProtocolVersion: 2}}}, {Body: &agentv1.AgentFrame_Report{Report: &agentv1.Report{}}}} {
		st := connect(t, c, tokenA, false)
		_ = st.Send(f)
		code(t, st, codes.InvalidArgument)
	}
	st2 := connect(t, c, tokenA, true)
	_ = recv(t, st2)
	_ = st2.Send(&agentv1.AgentFrame{Body: &agentv1.AgentFrame_Hello{Hello: &agentv1.Hello{ProtocolVersion: 1}}})
	code(t, st2, codes.InvalidArgument)
	for _, r := range []*agentv1.Report{{Revision: 999, Phase: agentv1.Report_APPLIED, ObservedAt: time.Now().Format(time.RFC3339Nano)}, {Revision: 1, Sha256: []byte("bad"), Phase: agentv1.Report_APPLIED, ObservedAt: "bad"}} {
		st := connect(t, c, tokenA, true)
		_ = recv(t, st)
		_ = st.Send(&agentv1.AgentFrame{Body: &agentv1.AgentFrame_Report{Report: r}})
		code(t, st, codes.InvalidArgument)
	}
}
func TestCredentialReplacementIsAtomicAndRevokesExistingStreams(t *testing.T) {
	s, c := setup(t, limits())
	_ = s.Publish(candidate(t, false), 1)
	st := connect(t, c, tokenA, true)
	_ = recv(t, st)
	for _, bad := range [][]Credential{{credential("one", "agent-a", tokenA), credential("two", "agent-b", tokenA)}, {{ID: "empty", AgentRef: "agent-a"}}, {credential("one", "bad agent", tokenB)}} {
		if e := s.ReplaceCredentials(bad); e == nil {
			t.Fatal("invalid replacement accepted")
		}
	}
	// The old stream still works after an invalid transaction.
	_ = s.Publish(candidate(t, false), 2)
	x := recv(t, st)
	if x.Revision != 2 {
		t.Fatal(x)
	}
	if e := s.ReplaceCredentials([]Credential{credential("one", "agent-a", tokenB)}); e != nil {
		t.Fatal(e)
	}
	code(t, st, codes.Unauthenticated)
	code(t, connect(t, c, tokenA, false), codes.Unauthenticated)
	rotated := connect(t, c, tokenB, true)
	_ = recv(t, rotated)
	if e := s.ReplaceCredentials([]Credential{credential("one", "agent-b", tokenB)}); e != nil {
		t.Fatal(e)
	}
	code(t, rotated, codes.Unauthenticated)
	rebound := connect(t, c, tokenB, true)
	if x := recv(t, rebound); x.AgentRef != "agent-b" || len(x.Targets) != 0 {
		t.Fatal(x)
	}
	_ = s.ReplaceCredentials(nil)
	code(t, rebound, codes.Unauthenticated)
	code(t, connect(t, c, tokenB, false), codes.Unauthenticated)
}
func TestPublicationIsAtomicAndRemovesAbsentAgent(t *testing.T) {
	s, c := setup(t, limits())
	c1 := candidate(t, false)
	if e := s.Publish(c1, 2); e != nil {
		t.Fatal(e)
	}
	for _, r := range []uint64{0, 1, 2} {
		if e := s.Publish(c1, r); e == nil {
			t.Fatal("rollback accepted")
		}
	}
	if e := s.Publish(nil, 3); e == nil {
		t.Fatal("nil accepted")
	}
	st := connect(t, c, tokenA, true)
	if x := recv(t, st); x.Revision != 2 {
		t.Fatal(x)
	}
	// Empty candidate with no Agent facts still withdraws previously enrolled scope.
	data, _ := os.ReadFile("../../examples/static/empty.yaml")
	b, _ := configuration.Decode(data)
	empty, e := configuration.Compile(b, configuration.Facts{AuthorityID: authority})
	if e != nil {
		t.Fatal(e)
	}
	if e = s.Publish(empty, 3); e != nil {
		t.Fatal(e)
	}
	if x := recv(t, st); x.Revision != 3 || len(x.Targets) != 0 {
		t.Fatal(x)
	}
}
func TestBoundsTimeoutsAndClose(t *testing.T) {
	for _, l := range []Limits{{}, func() Limits { x := limits(); x.MaxSnapshotBytes = (4 << 20) + 1; return x }()} {
		if s, e := New(authority, l); e == nil || s != nil {
			t.Fatal("bad limits accepted")
		}
	}
	t.Run("hello", func(t *testing.T) {
		l := limits()
		l.HelloTimeout = 30 * time.Millisecond
		_, c := setup(t, l)
		code(t, connect(t, c, tokenA, false), codes.DeadlineExceeded)
	})
	t.Run("streams", func(t *testing.T) {
		l := limits()
		l.MaxStreams = 1
		s, c := setup(t, l)
		_ = s.Publish(candidate(t, false), 1)
		_ = s.ReplaceCredentials([]Credential{credential("one", "agent-a", tokenA), credential("two", "agent-b", tokenB)})
		a := connect(t, c, tokenA, true)
		_ = recv(t, a)
		code(t, connect(t, c, tokenB, false), codes.ResourceExhausted)
		b := connect(t, c, tokenA, true)
		_ = recv(t, b)
		code(t, a, codes.Aborted)
	})
	t.Run("establishment", func(t *testing.T) {
		l := limits()
		l.EstablishPerSecond = 1
		_, c := setup(t, l)
		code(t, connect(t, c, "bad", false), codes.Unauthenticated)
		code(t, connect(t, c, tokenA, false), codes.ResourceExhausted)
	})
	t.Run("messages", func(t *testing.T) {
		l := limits()
		l.MessagesPerSecond = 1
		s, c := setup(t, l)
		_ = s.Publish(candidate(t, false), 1)
		st := connect(t, c, tokenA, true)
		x := recv(t, st)
		sendReport(t, st, x)
		code(t, st, codes.ResourceExhausted)
	})
	t.Run("snapshot", func(t *testing.T) {
		l := limits()
		l.MaxSnapshotBytes = 100
		s, _ := setup(t, l)
		if e := s.Publish(candidate(t, false), 1); e == nil {
			t.Fatal("oversize accepted")
		}
		if e := s.Publish(candidate(t, true), 1); e != nil {
			t.Fatal(e)
		}
	})
	t.Run("credentials", func(t *testing.T) {
		l := limits()
		l.MaxCredentials = 1
		s, _ := setup(t, l)
		if e := s.ReplaceCredentials([]Credential{credential("1", "a", tokenA), credential("2", "b", tokenB)}); e == nil {
			t.Fatal("credential limit ignored")
		}
	})
	t.Run("close", func(t *testing.T) {
		s, c := setup(t, limits())
		_ = s.Publish(candidate(t, false), 1)
		st := connect(t, c, tokenA, true)
		_ = recv(t, st)
		s.Close()
		s.Close()
		code(t, st, codes.Unavailable)
		code(t, connect(t, c, tokenA, false), codes.Unavailable)
		if s.Publish(candidate(t, true), 2) == nil || s.ReplaceCredentials(nil) == nil {
			t.Fatal("closed mutation accepted")
		}
	})
}
func TestVerifiedServerTLS(t *testing.T) {
	key, e := rsa.GenerateKey(rand.Reader, 2048)
	if e != nil {
		t.Fatal(e)
	}
	cert := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "controller.test"}, DNSNames: []string{"controller.test"}, NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour), KeyUsage: x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}, IsCA: true, BasicConstraintsValid: true}
	der, e := x509.CreateCertificate(rand.Reader, cert, cert, &key.PublicKey, key)
	if e != nil {
		t.Fatal(e)
	}
	pair, e := tls.X509KeyPair(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}))
	if e != nil {
		t.Fatal(e)
	}
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
	s, _ := setup(t, limits())
	_ = s.Publish(candidate(t, false), 1)
	srv := &tls.Config{MinVersion: tls.VersionTLS12, Certificates: []tls.Certificate{pair}}
	good := client(t, s, srv, &tls.Config{MinVersion: tls.VersionTLS12, RootCAs: pool, ServerName: "controller.test"})
	_ = recv(t, connect(t, good, tokenA, true))
	for _, cfg := range []*tls.Config{{MinVersion: tls.VersionTLS12, RootCAs: x509.NewCertPool(), ServerName: "controller.test"}, {MinVersion: tls.VersionTLS12, RootCAs: pool, ServerName: "wrong.test"}} {
		bad := client(t, s, srv, cfg)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		st, e := bad.Connect(ctx)
		if e == nil {
			_, e = st.Recv()
		}
		cancel()
		if e == nil {
			t.Fatal("TLS verification bypassed")
		}
	}
}

// Wire events observe the real gRPC transport; no fake Stream or Send is used.
type payloadEvents struct{ sent chan uint64 }

func (p *payloadEvents) TagRPC(ctx context.Context, _ *stats.RPCTagInfo) context.Context { return ctx }
func (p *payloadEvents) HandleRPC(_ context.Context, v stats.RPCStats) {
	if x, ok := v.(*stats.OutPayload); ok {
		if f, ok := x.Payload.(*agentv1.ControllerFrame); ok {
			p.sent <- f.Snapshot.Revision
		}
	}
}
func (p *payloadEvents) TagConn(ctx context.Context, _ *stats.ConnTagInfo) context.Context {
	return ctx
}
func (p *payloadEvents) HandleConn(context.Context, stats.ConnStats) {}
func largeCandidate(t *testing.T) *configuration.Candidate {
	t.Helper()
	data, e := os.ReadFile("../../examples/static/gitea.yaml")
	if e != nil {
		t.Fatal(e)
	}
	parts := strings.Split(string(data), "---\n")
	grant := parts[1]
	for i := 0; i < 2500; i++ {
		data = append(data, []byte("---\n"+strings.ReplaceAll(grant, "browser", fmt.Sprintf("extra-%04d-%s", i, strings.Repeat("x", 80))))...)
	}
	raw, _ := os.ReadFile("../../examples/static/facts.json")
	var f configuration.Facts
	_ = json.Unmarshal(raw, &f)
	b, e := configuration.Decode(data)
	if e != nil {
		t.Fatal(e)
	}
	c, e := configuration.Compile(b, f)
	if e != nil {
		t.Fatal(e)
	}
	return c
}
func waitInactive(t *testing.T, s *Service) {
	t.Helper()
	until := time.Now().Add(time.Second)
	for time.Now().Before(until) {
		s.mu.Lock()
		n := len(s.sessions)
		s.mu.Unlock()
		if n == 0 {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("stream slot retained")
}
func drainCode(t *testing.T, st grpc.BidiStreamingClient[agentv1.AgentFrame, agentv1.ControllerFrame], want codes.Code) {
	t.Helper()
	for i := 0; i < 4; i++ {
		_, e := st.Recv()
		if e != nil {
			if status.Code(e) != want {
				t.Fatalf("got %v want %v", e, want)
			}
			return
		}
	}
	t.Fatal("unbounded queued snapshots")
}
func TestSlowStreamRevocationTimeoutAndCoalescing(t *testing.T) {
	large := largeCandidate(t)
	for _, mode := range []string{"revoke", "timeout", "coalesce"} {
		t.Run(mode, func(t *testing.T) {
			l := limits()
			if mode == "timeout" {
				l.SendTimeout = 40 * time.Millisecond
			} else {
				l.SendTimeout = 30 * time.Second
			}
			s, unused := setup(t, l)
			_ = unused
			ev := &payloadEvents{sent: make(chan uint64, 10)}
			c := client(t, s, nil, nil, grpc.StatsHandler(ev))
			if e := s.Publish(large, 1); e != nil {
				t.Fatal(e)
			}
			st := connect(t, c, tokenA, true)
			select {
			case r := <-ev.sent:
				if r != 1 {
					t.Fatal(r)
				}
			case <-time.After(time.Second):
				t.Fatal("first send not observed")
			}
			// No Recv: the first large message fills the HTTP/2 window. The second send
			// exhausts write quota; updates after it must collapse to the latest state.
			if e := s.Publish(large, 2); e != nil {
				t.Fatal(e)
			}
			if mode == "revoke" {
				if e := s.ReplaceCredentials(nil); e != nil {
					t.Fatal(e)
				}
				waitInactive(t, s)
				drainCode(t, st, codes.Unauthenticated)
			}
			if mode == "timeout" {
				waitInactive(t, s)
				drainCode(t, st, codes.DeadlineExceeded)
			}
			if mode == "coalesce" {
				for r := uint64(3); r <= 8; r++ {
					if e := s.Publish(large, r); e != nil {
						t.Fatal(e)
					}
				}
				seen := []uint64{}
				for len(seen) < 4 {
					r := recv(t, st).Revision
					seen = append(seen, r)
					if r == 8 {
						break
					}
				}
				if seen[len(seen)-1] != 8 || len(seen) > 3 {
					t.Fatal("updates did not coalesce", seen)
				}
			}
		})
	}
}
func TestOldReportCannotOverwriteSuccessor(t *testing.T) {
	s, c := setup(t, limits())
	_ = s.Publish(candidate(t, false), 1)
	old := connect(t, c, tokenA, true)
	first := recv(t, old)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 30; i++ {
			e := old.Send(&agentv1.AgentFrame{Body: &agentv1.AgentFrame_Report{Report: &agentv1.Report{Revision: first.Revision, Sha256: first.Sha256, Phase: agentv1.Report_FAILED, ObservedAt: time.Now().Format(time.RFC3339Nano)}}})
			if e != nil {
				return
			}
		}
	}()
	next := connect(t, c, tokenA, true)
	snap := recv(t, next)
	sendReport(t, next, snap)
	wg.Wait()
	until := time.Now().Add(time.Second)
	for time.Now().Before(until) {
		o, ok := s.Report("agent-a")
		if ok && o.Connected && o.Report.Phase == agentv1.Report_APPLIED {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("successor report missing or overwritten")
}

func TestRejectRetargetedRetainedGrant(t *testing.T) {
	s, _ := setup(t, limits())
	old := candidate(t, false)
	if e := s.Publish(old, 1); e != nil {
		t.Fatal(e)
	}
	raw, _ := os.ReadFile("../../examples/static/facts.json")
	var f configuration.Facts
	_ = json.Unmarshal(raw, &f)
	data, _ := os.ReadFile("../../examples/static/gitea.yaml")
	// Change the physical port while retaining the Resource and Grant identities.
	data = bytes.Replace(data, []byte("port: 443"), []byte("port: 8443"), 1)
	b, e := configuration.Decode(data)
	if e != nil {
		t.Fatal(e)
	}
	changed, e := configuration.Compile(b, f)
	if e != nil {
		t.Fatal(e)
	}
	if e = s.Publish(changed, 2); e == nil {
		t.Fatal("retained Grant retargeted through publication")
	}
}

func TestTCPLoopbackStream(t *testing.T) {
	s, e := New(authority, limits())
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	if e = s.ReplaceCredentials([]Credential{credential("tcp", "agent-a", tokenA)}); e != nil {
		t.Fatal(e)
	}
	if e = s.Publish(candidate(t, false), 1); e != nil {
		t.Fatal(e)
	}
	listener, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	server := grpc.NewServer(grpc.MaxRecvMsgSize(MaxWireBytes), grpc.MaxSendMsgSize(MaxWireBytes))
	agentv1.RegisterAgentSyncServer(server, s)
	go func() { _ = server.Serve(listener) }()
	defer server.Stop()
	cc, e := grpc.NewClient(listener.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if e != nil {
		t.Fatal(e)
	}
	defer cc.Close()
	st := connect(t, agentv1.NewAgentSyncClient(cc), tokenA, true)
	snap := recv(t, st)
	sendReport(t, st, snap)
	_ = waitReport(t, s, "agent-a")
	if e = s.ReplaceCredentials(nil); e != nil {
		t.Fatal(e)
	}
	code(t, st, codes.Unauthenticated)
}

func TestPublicationPreflightDoesNotMutate(t *testing.T) {
	s, c := setup(t, limits())
	desired := candidate(t, false)
	if e := s.CheckPublication(desired, 1); e != nil {
		t.Fatal(e)
	}
	if s.revision != 0 || s.candidate != nil || len(s.snapshots) != 0 {
		t.Fatal("preflight published state")
	}
	if e := s.Publish(desired, 1); e != nil {
		t.Fatal(e)
	}
	st := connect(t, c, tokenA, true)
	if x := recv(t, st); x.Revision != 1 {
		t.Fatal(x)
	}
	if e := s.CheckPublication(desired, 1); e == nil {
		t.Fatal("preflight ignored revision guard")
	}
}
