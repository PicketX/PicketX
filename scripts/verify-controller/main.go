// SPDX-License-Identifier: Apache-2.0
// This build-time smoke client verifies a running artifact, never kernel apply.
package main

import (
	"context"
	"fmt"
	"github.com/PicketX/PicketX/pkg/agentsnapshot"
	agentv1 "github.com/PicketX/PicketX/pkg/protocol/agent/v1alpha1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"os"
	"time"
)

func main() {
	if e := run(); e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
}
func run() error {
	if len(os.Args) != 3 {
		return fmt.Errorf("address and private token file required")
	}
	token, e := os.ReadFile(os.Args[2])
	if e != nil {
		return e
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+string(token))
	cc, e := grpc.NewClient(os.Args[1], grpc.WithTransportCredentials(insecure.NewCredentials()))
	if e != nil {
		return e
	}
	defer cc.Close()
	st, e := agentv1.NewAgentSyncClient(cc).Connect(ctx)
	if e != nil {
		return e
	}
	if e = st.Send(&agentv1.AgentFrame{Body: &agentv1.AgentFrame_Hello{Hello: &agentv1.Hello{ProtocolVersion: 1}}}); e != nil {
		return e
	}
	frame, e := st.Recv()
	if e != nil {
		return e
	}
	s := frame.GetSnapshot()
	if e = agentsnapshot.Verify(s); e != nil {
		return e
	}
	if s.AuthorityId != "static:example-v1" || s.AgentRef != "agent-a" || s.Revision != 1 || len(s.Targets) != 2 {
		return fmt.Errorf("unexpected static Controller snapshot")
	}
	if e = st.Send(&agentv1.AgentFrame{Body: &agentv1.AgentFrame_Report{Report: &agentv1.Report{Revision: s.Revision, Sha256: s.Sha256, Phase: agentv1.Report_RECEIVED, ObservedAt: time.Now().UTC().Format(time.RFC3339Nano)}}}); e != nil {
		return e
	}
	fmt.Println("static gRPC received revision=1 targets=2; no enforcement claim")
	return nil
}
