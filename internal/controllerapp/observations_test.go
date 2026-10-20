// SPDX-License-Identifier: Apache-2.0
package controllerapp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/PicketX/PicketX/internal/agentsync"
	"github.com/PicketX/PicketX/internal/cli"
	"github.com/PicketX/PicketX/internal/controllerclient"
	agentv1 "github.com/PicketX/PicketX/pkg/protocol/agent/v1alpha1"
)

func TestControllerDiagnosticsAcceptedEscapingPublication(t *testing.T) {
	c := managedConfig(t)
	p := start(t, c)
	ready(t, p, 200)
	policy := "apiVersion: picketx.io/v1alpha1\nkind: Resource\nmetadata: {name: large, uid: large-v1}\nspec:\n endpoints:\n - name: web\n   network: {protocol: TCP, port: 9443}\n#" + strings.Repeat("<", 3<<20) + "\n"
	facts := fmt.Sprintf(`{"authorityID":%q,"agents":[],"contexts":[]}`, c.Agent.Authority)
	var input bytes.Buffer
	encoder := json.NewEncoder(&input)
	encoder.SetEscapeHTML(false) // A supported client need not HTML-escape JSON strings.
	if err := encoder.Encode(map[string]string{"policy": policy, "facts": facts, "reason": "diagnose accepted escaping publication"}); err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest("PUT", "http://"+p.http+"/api/v1/publication", &input)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("If-Match", `"1"`)
	transport := &http.Transport{Proxy: nil}
	defer transport.CloseIdleConnections()
	response, err := (&http.Client{Transport: transport, Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(response.Body)
	response.Body.Close()
	if err != nil || response.StatusCode != 200 || len(body) <= 16<<20 {
		t.Fatal("accepted HTML-escaping publication not exercised", response.StatusCode, len(body), err)
	}
	key := filepath.Join(t.TempDir(), "management-key")
	write(t, key, []byte(adminToken))
	for _, name := range []string{"status", "doctor"} {
		var out, diag bytes.Buffer
		command := cli.New(&out, &diag)
		command.SetArgs([]string{name, "--controller", "http://" + p.http, "--key-file", key, "--format", "json", "--timeout", "30s"})
		err := command.Execute()
		var report controllerclient.Report
		if json.Unmarshal(out.Bytes(), &report) != nil || report.Revision != "2" || report.EnforcementConfirmed || (err == nil) != (name == "status") {
			t.Fatal("supported publication cannot be diagnosed", name, err)
		}
		if bytes.Contains(out.Bytes(), []byte(policy)) || bytes.Contains(out.Bytes(), []byte(adminToken)) {
			t.Fatal("publication diagnostic disclosed private input")
		}
	}
}

// Actual gRPC reports are assertions, not a substitute for Linux kernel tests.
func TestAgentObservationsRuntime(t *testing.T) {
	c := managedConfig(t)
	p := start(t, c)
	ready(t, p, 200)
	keyFile := filepath.Join(t.TempDir(), "management-key")
	write(t, keyFile, []byte(adminToken))
	checkCLI := func(want string, healthy bool) {
		t.Helper()
		var out, diagnostics bytes.Buffer
		command := cli.New(&out, &diagnostics)
		command.SetArgs([]string{"doctor", "--controller", "http://" + p.http, "--key-file", keyFile, "--format", "json"})
		err := command.Execute()
		if (err == nil) != healthy {
			t.Fatalf("CLI health mismatch: %v", err)
		}
		var report controllerclient.Report
		if json.Unmarshal(out.Bytes(), &report) != nil || len(report.Nodes) != 1 || report.Nodes[0].State != want || report.EnforcementConfirmed {
			t.Fatalf("incorrect actual Controller diagnostic: %s", out.String())
		}
		if bytes.Contains(out.Bytes(), []byte(adminToken)) {
			t.Fatal("CLI exposed management key")
		}
	}
	read := func() agentsync.Observations {
		t.Helper()
		code, h, data := apiCall(t, p, "GET", "/api/v1/agent-observations", nil, "", adminToken)
		if code != 200 || h.Get("Cache-Control") != "no-store" {
			t.Fatalf("observations %d %s", code, data)
		}
		var v agentsync.Observations
		if e := json.Unmarshal(data, &v); e != nil {
			t.Fatal(e)
		}
		return v
	}
	v := read()
	if len(v.Nodes) != 1 || v.Nodes[0].Freshness != "unknown" || v.Nodes[0].Connection != "disconnected" {
		t.Fatal(v)
	}
	checkCLI("disconnected", false)
	for _, tc := range []struct {
		token, path string
		code        int
	}{{"", "/api/v1/agent-observations", 401}, {adminToken, "/api/v1/agent-observations?x=1", 400}} {
		code, _, _ := apiCall(t, p, "GET", tc.path, nil, "", tc.token)
		if code != tc.code {
			t.Fatal(code, tc)
		}
	}
	st := stream(t, p)
	snap := snapshot(t, st)
	if e := st.Send(&agentv1.AgentFrame{Body: &agentv1.AgentFrame_Report{Report: &agentv1.Report{Revision: snap.Revision, Sha256: snap.Sha256, Phase: agentv1.Report_APPLIED, ObservedAt: "2099-01-01T00:00:00Z"}}}); e != nil {
		t.Fatal(e)
	}
	wait := func(check func(agentsync.NodeObservation) bool) agentsync.NodeObservation {
		t.Helper()
		deadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(deadline) {
			v := read()
			if len(v.Nodes) == 1 && check(v.Nodes[0]) {
				return v.Nodes[0]
			}
			time.Sleep(time.Millisecond * 5)
		}
		t.Fatal("observation condition timed out")
		return agentsync.NodeObservation{}
	}
	n := wait(func(n agentsync.NodeObservation) bool {
		return n.Report != nil && n.Report.Phase == "APPLIED" && n.Freshness == "fresh" && n.MatchesDesired
	})
	if n.Report.ObservedAt != "2099-01-01T00:00:00Z" || n.Report.ReceivedAt.IsZero() {
		t.Fatal(n)
	}
	checkCLI("reported-applied", true)
	policy, _ := os.ReadFile("../../examples/static/gitea.yaml")
	facts, _ := os.ReadFile("../../examples/static/facts.json")
	body, _ := json.Marshal(map[string]string{"policy": string(policy), "facts": string(facts), "reason": "observe pending delivery"})
	if code, _, data := apiCall(t, p, "PUT", "/api/v1/publication", body, `"1"`, adminToken); code != 200 {
		t.Fatal(code, string(data))
	}
	snapshot(t, st)
	n = wait(func(n agentsync.NodeObservation) bool { return n.DesiredRevision == "2" && !n.MatchesDesired })
	if n.Report.Revision != "1" {
		t.Fatal(n)
	}
	checkCLI("applying", false)
	if e := st.CloseSend(); e != nil {
		t.Fatal(e)
	}
	wait(func(n agentsync.NodeObservation) bool {
		return n.Connection == "disconnected" && n.Freshness == "stale"
	})
	checkCLI("disconnected", false)
	next := stream(t, p)
	snapshot(t, next)
	wait(func(n agentsync.NodeObservation) bool { return n.Connection == "connected" && n.Freshness == "stale" })
	checkCLI("stale", false)
}

func TestAgentObservationsRejectUnreadyPublisher(t *testing.T) {
	p := &publisher{}
	if _, e := p.AgentObservations(); e == nil {
		t.Fatal("unready served status")
	}
	p.ready.Store(true)
	p.failed = true
	if _, e := p.AgentObservations(); e == nil {
		t.Fatal("failed delivery served status")
	}
}
