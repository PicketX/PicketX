// SPDX-License-Identifier: Apache-2.0
package controllerapp

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/PicketX/PicketX/internal/agentsync"
	"github.com/PicketX/PicketX/internal/publication"
	"github.com/PicketX/PicketX/internal/runtimeconfig"
	"github.com/PicketX/PicketX/pkg/configuration"
)

const adminToken = "independent-management-runtime-token-1234567890"

func managedConfig(t *testing.T) runtimeconfig.Config {
	t.Helper()
	c := fixture(t)
	c.Source.Mode = "database"
	c.Source.Policy = ""
	c.Source.Facts = ""
	c.Management.Credentials = filepath.Join(t.TempDir(), "principals.json")
	sum := sha256.Sum256([]byte(adminToken))
	data, _ := json.Marshal(map[string]any{"principals": []map[string]any{{"id": "principal:admin", "sha256": hex.EncodeToString(sum[:]), "roles": []string{"role:admin"}}}})
	write(t, c.Management.Credentials, data)
	return c
}

type failureAtRevision struct {
	*agentsync.Service
	revision uint64
	err      error
}

func (f *failureAtRevision) Publish(c *configuration.Candidate, r uint64) error {
	if r == f.revision {
		return f.err
	}
	return f.Service.Publish(c, r)
}
func TestManagedPostCommitFailureStopsAndRecovers(t *testing.T) {
	c := managedConfig(t)
	boom := errors.New("post-commit distribution failed")
	p := startWithService(t, c, func(a string, l agentsync.Limits) (AgentService, error) {
		s, e := agentsync.New(a, l)
		if e != nil {
			return nil, e
		}
		return &failureAtRevision{Service: s, revision: 2, err: boom}, nil
	})
	ready(t, p, 200)
	policy, _ := os.ReadFile("../../examples/static/gitea.yaml")
	facts, _ := os.ReadFile("../../examples/static/facts.json")
	body, _ := json.Marshal(map[string]string{"policy": string(policy), "facts": string(facts), "reason": "durable before distribution"})
	code, _, data := apiCall(t, p, "PUT", "/api/v1/publication", body, `"1"`, adminToken)
	if code != 503 {
		t.Fatal(code, string(data))
	}
	select {
	case e := <-p.done:
		if !errors.Is(e, boom) {
			t.Fatal(e)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("distribution failure did not stop runtime")
	}
	p.cancel()
	p.cancel = nil
	p = start(t, c)
	ready(t, p, 200)
	s := snapshot(t, stream(t, p))
	if s.Revision != 2 || len(s.Targets) != 2 {
		t.Fatal("durable state not recovered", s)
	}
	code, _, data = apiCall(t, p, "GET", "/api/v1/audit", nil, "", adminToken)
	if code != 200 || !bytes.Contains(data, []byte("durable before distribution")) {
		t.Fatal(code, string(data))
	}
}
func apiCall(t *testing.T, p *process, method, path string, body []byte, match, auth string) (int, http.Header, []byte) {
	t.Helper()
	client := &http.Client{Timeout: time.Second, Transport: &http.Transport{Proxy: nil}}
	defer client.CloseIdleConnections()
	req, e := http.NewRequest(method, "http://"+p.http+path, bytes.NewReader(body))
	if e != nil {
		t.Fatal(e)
	}
	req.Header.Set("Authorization", "Bearer "+auth)
	req.Header.Set("Content-Type", "application/json")
	if match != "" {
		req.Header.Set("If-Match", match)
	}
	response, e := client.Do(req)
	if e != nil {
		t.Fatal(e)
	}
	defer response.Body.Close()
	data, e := io.ReadAll(response.Body)
	if e != nil {
		t.Fatal(e)
	}
	return response.StatusCode, response.Header, data
}
func TestManagedRuntimeHTTPDurabilityAndGRPC(t *testing.T) {
	c := managedConfig(t)
	p := start(t, c)
	ready(t, p, 200)
	first := snapshot(t, stream(t, p))
	if first.Revision != 1 || len(first.Targets) != 0 {
		t.Fatal("managed boot not explicit empty", first)
	}
	// Agent Token cannot become a management principal.
	if code, _, _ := apiCall(t, p, "GET", "/api/v1/publication", nil, "", token); code != 401 {
		t.Fatal("Agent Token accepted by management", code)
	}
	policy, _ := os.ReadFile("../../examples/static/gitea.yaml")
	facts, _ := os.ReadFile("../../examples/static/facts.json")
	body, _ := json.Marshal(map[string]string{"policy": string(policy), "facts": string(facts), "reason": "activate managed test inventory"})
	st := stream(t, p)
	_ = snapshot(t, st)
	code, header, data := apiCall(t, p, "PUT", "/api/v1/publication", body, `"1"`, adminToken)
	if code != 200 || header.Get("ETag") != `"2"` {
		t.Fatal(code, string(data))
	}
	accepted := snapshot(t, st)
	if accepted.Revision != 2 || len(accepted.Targets) != 2 {
		t.Fatal(accepted)
	}
	// Several credential polls cannot restore any file policy or consume a revision.
	time.Sleep(4 * c.Poll)
	code, _, data = apiCall(t, p, "GET", "/api/v1/publication", nil, "", adminToken)
	var doc publication.Document
	if e := json.Unmarshal(data, &doc); code != 200 || e != nil || doc.Revision != 2 || doc.Policy != string(policy) {
		t.Fatal(code, doc, e)
	}
	if code, _, _ = apiCall(t, p, "PUT", "/api/v1/publication", body, `"1"`, adminToken); code != 412 {
		t.Fatal(code)
	}
	p.stop(t)
	p = start(t, c)
	ready(t, p, 200)
	recovered := snapshot(t, stream(t, p))
	if recovered.Revision != 2 || !bytes.Equal(recovered.Sha256, accepted.Sha256) {
		t.Fatal("restart changed managed state", recovered)
	}
	write(t, c.Management.Credentials, []byte(`{"principals":[]}`))
	if code, _, _ = apiCall(t, p, "GET", "/api/v1/publication", nil, "", adminToken); code != 401 {
		t.Fatal("revoked manager accepted", code)
	}
	// Agent credentials remain independently revocable, even with no policy files.
	agent := stream(t, p)
	_ = snapshot(t, agent)
	write(t, c.Credentials, []byte(`{"credentials":[]}`))
	if _, e := agent.Recv(); e == nil {
		t.Fatal("Agent revocation not polled")
	}
}
