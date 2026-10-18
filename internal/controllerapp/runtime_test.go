// SPDX-License-Identifier: Apache-2.0
package controllerapp

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/PicketX/PicketX/internal/agentsync"
	"github.com/PicketX/PicketX/internal/publication"
	"github.com/PicketX/PicketX/internal/runtimeconfig"
	"github.com/PicketX/PicketX/pkg/configuration"
	agentv1 "github.com/PicketX/PicketX/pkg/protocol/agent/v1alpha1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

const token = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func write(t *testing.T, p string, data []byte) {
	t.Helper()
	tmp := p + ".tmp"
	if e := os.WriteFile(tmp, data, 0600); e != nil {
		t.Fatal(e)
	}
	if e := os.Rename(tmp, p); e != nil {
		t.Fatal(e)
	}
}
func fixture(t *testing.T) runtimeconfig.Config {
	t.Helper()
	dir := t.TempDir()
	p, e := os.ReadFile("../../examples/static/gitea.yaml")
	if e != nil {
		t.Fatal(e)
	}
	f, e := os.ReadFile("../../examples/static/facts.json")
	if e != nil {
		t.Fatal(e)
	}
	write(t, filepath.Join(dir, "policy.yaml"), p)
	write(t, filepath.Join(dir, "facts.json"), f)
	sum := sha256.Sum256([]byte(token))
	creds, _ := json.Marshal(map[string]any{"credentials": []map[string]string{{"id": "one", "agent_ref": "agent-a", "sha256": hex.EncodeToString(sum[:])}}})
	write(t, filepath.Join(dir, "credentials.json"), creds)
	c, e := runtimeconfig.Load("", nil, map[string]string{"http.listen": "127.0.0.1:0", "agent.enabled": "true", "agent.listen": "127.0.0.1:0", "agent.authority": "static:example-v1", "source.policy": filepath.Join(dir, "policy.yaml"), "source.facts": filepath.Join(dir, "facts.json"), "database": filepath.Join(dir, "state", "controller.sqlite"), "credentials": filepath.Join(dir, "credentials.json"), "poll": "15ms", "shutdown": "200ms"})
	if e != nil {
		t.Fatal(e)
	}
	return c
}

type process struct {
	http, agent string
	cancel      context.CancelFunc
	done        chan error
}

func start(t *testing.T, c runtimeconfig.Config) *process {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	p := &process{cancel: cancel, done: make(chan error, 1)}
	addrs := make(chan string, 2)
	deps := Dependencies{Listen: func(network, addr string) (net.Listener, error) {
		l, e := net.Listen(network, addr)
		if e == nil {
			addrs <- l.Addr().String()
		}
		return l, e
	}}
	go func() { p.done <- Run(ctx, c, slog.New(slog.NewTextHandler(io.Discard, nil)), deps) }()
	for _, target := range []*string{&p.http, &p.agent} {
		if !c.Agent.Enabled && target == &p.agent {
			continue
		}
		select {
		case *target = <-addrs:
		case e := <-p.done:
			cancel()
			t.Fatal("startup failed", e)
		case <-time.After(3 * time.Second):
			cancel()
			t.Fatal("listener startup timeout")
		}
	}
	t.Cleanup(func() { p.stop(t) })
	return p
}
func (p *process) stop(t *testing.T) {
	t.Helper()
	if p.cancel == nil {
		return
	}
	p.cancel()
	p.cancel = nil
	select {
	case e := <-p.done:
		if e != nil {
			t.Error(e)
		}
	case <-time.After(3 * time.Second):
		t.Error("runtime did not stop")
	}
}
func ready(t *testing.T, p *process, want int) {
	t.Helper()
	client := &http.Client{Timeout: 200 * time.Millisecond, Transport: &http.Transport{Proxy: nil}}
	defer client.CloseIdleConnections()
	until := time.Now().Add(3 * time.Second)
	for time.Now().Before(until) {
		r, e := client.Get("http://" + p.http + "/readyz")
		if e == nil {
			_, _ = io.Copy(io.Discard, r.Body)
			_ = r.Body.Close()
			if r.StatusCode == want {
				return
			}
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("readiness never became %d", want)
}
func stream(t *testing.T, p *process) grpc.BidiStreamingClient[agentv1.AgentFrame, agentv1.ControllerFrame] {
	t.Helper()
	cc, e := grpc.NewClient(p.agent, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = cc.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+token)
	st, e := agentv1.NewAgentSyncClient(cc).Connect(ctx)
	if e != nil {
		t.Fatal(e)
	}
	if e = st.Send(&agentv1.AgentFrame{Body: &agentv1.AgentFrame_Hello{Hello: &agentv1.Hello{ProtocolVersion: 1}}}); e != nil {
		t.Fatal(e)
	}
	return st
}
func snapshot(t *testing.T, st grpc.BidiStreamingClient[agentv1.AgentFrame, agentv1.ControllerFrame]) *agentv1.Snapshot {
	t.Helper()
	m, e := st.Recv()
	if e != nil {
		t.Fatal(e)
	}
	return m.Snapshot
}
func TestRuntimeTCPRestartReloadAndRevocation(t *testing.T) {
	c := fixture(t)
	p := start(t, c)
	ready(t, p, 200)
	st := stream(t, p)
	first := snapshot(t, st)
	if first.Revision != 1 || len(first.Targets) != 2 {
		t.Fatal(first)
	}
	p.stop(t)
	write(t, c.Source.Policy, []byte("invalid"))
	p = start(t, c)
	ready(t, p, 503)
	st = stream(t, p)
	recovered := snapshot(t, st)
	if recovered.Revision != 1 || !bytes.Equal(recovered.Sha256, first.Sha256) {
		t.Fatal("restart renewed/replaced state", recovered)
	}
	// A valid credential revocation must still commit while policy is rejected.
	write(t, c.Credentials, []byte(`{"credentials":[]}`))
	_, e := st.Recv()
	if status.Code(e) != codes.Unauthenticated {
		t.Fatal(e)
	}
	policy, _ := os.ReadFile("../../examples/static/gitea.yaml")
	write(t, c.Source.Policy, policy)
	ready(t, p, 200)
}
func TestNewCredentialReceivesPrepublishedNonemptyInventory(t *testing.T) {
	c := fixture(t)
	raw, _ := os.ReadFile(c.Source.Facts)
	var document map[string]any
	if e := json.Unmarshal(raw, &document); e != nil {
		t.Fatal(e)
	}
	agents := document["agents"].([]any)
	copyAgent := map[string]any{}
	for k, v := range agents[0].(map[string]any) {
		copyAgent[k] = v
	}
	copyAgent["Name"] = "agent-b"
	document["agents"] = append(agents, copyAgent)
	contexts := document["contexts"].([]any)
	copyContext := map[string]any{}
	for k, v := range contexts[0].(map[string]any) {
		copyContext[k] = v
	}
	copyContext["AgentRef"] = "agent-b"
	document["contexts"] = append(contexts, copyContext)
	raw, _ = json.Marshal(document)
	f, e := configuration.DecodeFacts(raw)
	if e != nil {
		t.Fatal("fixture facts", e)
	}
	write(t, c.Source.Facts, raw)
	policy, _ := os.ReadFile(c.Source.Policy)
	policy = bytes.Replace(policy, []byte("    - agentRef: {name: agent-a}"), []byte("    - agentRef: {name: agent-a}\n    - agentRef: {name: agent-b}"), 1)
	b, e := configuration.Decode(policy)
	if e != nil {
		t.Fatal("fixture policy", e)
	}
	if _, e = configuration.Compile(b, f); e != nil {
		t.Fatal("fixture compile", e)
	}
	write(t, c.Source.Policy, policy)
	write(t, c.Credentials, []byte(`{"credentials":[]}`))
	p := start(t, c)
	ready(t, p, 200)
	sum := sha256.Sum256([]byte(token))
	raw, _ = json.Marshal(map[string]any{"credentials": []map[string]string{{"id": "b", "agent_ref": "agent-b", "sha256": hex.EncodeToString(sum[:])}}})
	write(t, c.Credentials, raw)
	// Wait for the polling owner to load the credential set. Failed admission is
	// harmless and rate-limited; retry uses a fresh connection, not test hooks.
	until := time.Now().Add(3 * time.Second)
	for time.Now().Before(until) {
		st := stream(t, p)
		m, e := st.Recv()
		if e == nil {
			if m.Snapshot.AgentRef != "agent-b" || len(m.Snapshot.Targets) != 2 || m.Snapshot.Revision != 1 {
				t.Fatal(m)
			}
			return
		}
		time.Sleep(15 * time.Millisecond)
	}
	t.Fatal("new scope never received current nonempty intent")
}
func TestStartupFailureClosesReservedListeners(t *testing.T) {
	c := fixture(t)
	var reserved net.Listener
	calls := 0
	boom := errors.New("listen failed")
	e := Run(context.Background(), c, slog.New(slog.NewTextHandler(io.Discard, nil)), Dependencies{Listen: func(n, a string) (net.Listener, error) {
		calls++
		if calls == 2 {
			return nil, boom
		}
		l, e := net.Listen(n, a)
		reserved = l
		return l, e
	}})
	if !errors.Is(e, boom) {
		t.Fatal(e)
	}
	if reserved == nil {
		t.Fatal("no reserved listener")
	}
	if _, e = net.DialTimeout("tcp", reserved.Addr().String(), 100*time.Millisecond); e == nil {
		t.Fatal("listener leaked")
	}
	c.Agent.TLSCert = "missing-cert"
	c.Agent.TLSKey = "missing-key"
	if e = Run(context.Background(), c, slog.Default(), Dependencies{}); e == nil {
		t.Fatal("TLS error ignored")
	}
}
func TestFirstInvalidSourceFailsAndDefaultHTTPRuns(t *testing.T) {
	c := fixture(t)
	write(t, c.Source.Policy, []byte("invalid"))
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if e := Run(ctx, c, slog.New(slog.NewTextHandler(io.Discard, nil)), Dependencies{}); e == nil {
		t.Fatal("invalid initial publication accepted")
	}
	d, e := runtimeconfig.Load("", nil, map[string]string{"http.listen": "127.0.0.1:0"})
	if e != nil {
		t.Fatal(e)
	}
	p := start(t, d)
	ready(t, p, 200)
}
func TestCredentialDecoderStrict(t *testing.T) {
	for _, data := range []string{`null`, `{"credentials":null}`, `{"credentials":[],"credentials":[]}`, `{"Credentials":[]}`, `{"credentials":[{"id":"x","agent_ref":"a","sha256":"bad"}]}`, `{"credentials":[]} {}`} {
		if _, e := decodeCredentials([]byte(data)); e == nil {
			t.Fatal("invalid credential input accepted")
		}
	}
}

type failedDistribution struct {
	*agentsync.Service
	err error
}

func (f *failedDistribution) Publish(*configuration.Candidate, uint64) error { return f.err }
func TestDistributionFailureStopsAfterDurableCommit(t *testing.T) {
	c := fixture(t)
	boom := errors.New("distribution failed")
	e := Run(context.Background(), c, slog.New(slog.NewTextHandler(io.Discard, nil)), Dependencies{NewService: func(a string, l agentsync.Limits) (AgentService, error) {
		s, e := agentsync.New(a, l)
		if e != nil {
			return nil, e
		}
		return &failedDistribution{Service: s, err: boom}, nil
	}})
	if !errors.Is(e, boom) {
		t.Fatal(e)
	}
	s, e := publication.Open(c.Database, c.Agent.Authority)
	if e != nil {
		t.Fatal(e)
	}
	_, revision := s.Current()
	_ = s.Close()
	if revision != 1 {
		t.Fatal("durable commit missing", revision)
	}
	p := start(t, c)
	ready(t, p, 200)
	if got := snapshot(t, stream(t, p)); got.Revision != 1 {
		t.Fatal("recovery bumped revision", got)
	}
}
