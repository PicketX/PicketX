// SPDX-License-Identifier: Apache-2.0
package cli

import (
	"bytes"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/PicketX/PicketX/pkg/configuration"
)

const diagnosticKey = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
const diagnosticHash = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func requireDiagnostics(t *testing.T) {
	t.Helper()
	c, _, e := New(&bytes.Buffer{}, &bytes.Buffer{}).Find([]string{"status"})
	if e != nil || c.Name() != "status" {
		t.Fatal("status command is missing")
	}
}
func diagnosticKeyFile(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "management-key")
	if e := os.WriteFile(p, []byte(diagnosticKey+"\n"), 0600); e != nil {
		t.Fatal(e)
	}
	return p
}
func diagnosticObservation() map[string]any {
	return map[string]any{"authority": "authority:test", "observed_at": time.Now().UTC(), "nodes": []any{
		map[string]any{"agent_ref": "node-a", "desired_revision": "1", "desired_sha256": diagnosticHash,
			"connection": "connected", "freshness": "fresh", "fresh_for_millis": 59000, "matches_desired": true,
			"report": map[string]any{"revision": "1", "sha256": diagnosticHash, "phase": "APPLIED", "observed_at": "2099-01-01T00:00:00Z", "received_at": time.Now().UTC()}},
	}}
}
func diagnosticServer(t *testing.T, observation map[string]any, hook func(http.ResponseWriter, *http.Request) bool) *httptest.Server {
	t.Helper()
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Error("diagnostics attempted mutation")
			w.WriteHeader(405)
			return
		}
		if r.URL.Path == "/readyz" {
			if r.Header.Get("Authorization") != "" {
				t.Error("credential sent to unauthenticated readiness endpoint")
			}
		} else if r.Header.Get("Authorization") != "Bearer "+diagnosticKey {
			w.WriteHeader(401)
			return
		}
		if hook != nil && hook(w, r) {
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/readyz":
			fmt.Fprint(w, `{"status":"ready"}`)
		case "/api/v1/publication":
			fmt.Fprint(w, `{"authority":"authority:test","revision":1,"policy":"PRIVATE-POLICY","facts":"PRIVATE-FACTS"}`)
		case "/api/v1/agent-observations":
			_ = json.NewEncoder(w).Encode(observation)
		default:
			t.Error("unexpected diagnostic path", r.URL.Path)
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(s.Close)
	return s
}
func runDiagnostic(t *testing.T, origin, key, command string, extra ...string) (map[string]any, string, error) {
	t.Helper()
	requireDiagnostics(t)
	args := append([]string{command, "--controller", origin, "--key-file", key, "--format", "json"}, extra...)
	out, e := execute(args...)
	for _, secret := range []string{diagnosticKey, "PRIVATE-POLICY", "PRIVATE-FACTS", "PRIVATE-ERROR"} {
		if strings.Contains(out, secret) || (e != nil && strings.Contains(e.Error(), secret)) {
			t.Fatal("diagnostics leaked private data")
		}
	}
	var v map[string]any
	if out != "" {
		if err := json.Unmarshal([]byte(out), &v); err != nil {
			t.Fatal("invalid report", out, err)
		}
	}
	return v, out, e
}

func TestControllerDiagnosticsReportedAppliedIsNotEnforcementProof(t *testing.T) {
	s := diagnosticServer(t, diagnosticObservation(), nil)
	v, _, e := runDiagnostic(t, s.URL, diagnosticKeyFile(t), "doctor")
	if e != nil {
		t.Fatal(e)
	}
	if v["scope"] != "controller-observations" || v["enforcement_confirmed"] != false || v["revision"] != "1" || v["ready"] != true {
		t.Fatal(v)
	}
	n := v["nodes"].([]any)[0].(map[string]any)
	if n["state"] != "reported-applied" || n["reported_revision"] != "1" {
		t.Fatal(n)
	}
}

func TestControllerDiagnosticsIPv6AndExactRevisionScope(t *testing.T) {
	const revision = "18446744073709551615"
	o := diagnosticObservation()
	n := o["nodes"].([]any)[0].(map[string]any)
	n["desired_revision"] = revision
	n["report"].(map[string]any)["revision"] = revision
	other := diagnosticObservation()["nodes"].([]any)[0].(map[string]any)
	other["agent_ref"] = "node-b"
	other["desired_revision"] = revision
	other["report"].(map[string]any)["revision"] = revision
	o["nodes"] = []any{other, n}
	s := diagnosticServer(t, o, func(w http.ResponseWriter, r *http.Request) bool {
		if r.URL.Path != "/api/v1/publication" {
			return false
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"authority":"authority:test","revision":%s,"policy":"","facts":""}`, revision)
		return true
	})
	listener, e := net.Listen("tcp6", "[::1]:0")
	if e != nil {
		t.Fatal(e)
	}
	v6 := httptest.NewUnstartedServer(s.Config.Handler)
	v6.Listener.Close()
	v6.Listener = listener
	v6.Start()
	t.Cleanup(v6.Close)
	key := diagnosticKeyFile(t)
	v, _, e := runDiagnostic(t, v6.URL, key, "doctor", "--agent", "node-a")
	if e != nil || v["revision"] != revision {
		t.Fatal("IPv6 selected scope or exact uint64 revision failed", v, e)
	}
	nodes := v["nodes"].([]any)
	if len(nodes) != 1 || nodes[0].(map[string]any)["agent_ref"] != "node-a" {
		t.Fatal("unselected scope returned", v)
	}
	v, _, e = runDiagnostic(t, v6.URL, key, "status")
	if e != nil || v["nodes"].([]any)[0].(map[string]any)["agent_ref"] != "node-a" {
		t.Fatal("scope order is unstable", v, e)
	}
}

func TestControllerDiagnosticsParseErrorsDoNotEchoPrivateArguments(t *testing.T) {
	for _, name := range []string{"status", "doctor"} {
		for _, extra := range [][]string{{"--timeout", "/private/management-key"}, {"--" + diagnosticKey}, {"/private/extra-path"}} {
			t.Run(name+extra[0], func(t *testing.T) {
				var out, diagnostics bytes.Buffer
				command := New(&out, &diagnostics)
				command.SetArgs(append([]string{name}, extra...))
				err := command.Execute()
				if err == nil || out.Len() != 0 {
					t.Fatal("invalid private arguments accepted")
				}
				// cmd/picketx prints the returned error to stderr after Execute.
				fmt.Fprintln(&diagnostics, err)
				for _, private := range []string{"/private/", diagnosticKey} {
					if strings.Contains(err.Error(), private) || strings.Contains(diagnostics.String(), private) {
						t.Fatal("command parsing echoed a private argument")
					}
				}
			})
		}
	}
}

func TestControllerDiagnosticsRejectsOverLimitPublicationFields(t *testing.T) {
	for _, field := range []string{"policy", "facts"} {
		t.Run(field, func(t *testing.T) {
			s := diagnosticServer(t, diagnosticObservation(), func(w http.ResponseWriter, r *http.Request) bool {
				if r.URL.Path != "/api/v1/publication" {
					return false
				}
				w.Header().Set("Content-Type", "application/json")
				v := map[string]any{"authority": "authority:test", "revision": 1, "policy": "", "facts": ""}
				v[field] = strings.Repeat("x", configuration.MaxBytes+1)
				_ = json.NewEncoder(w).Encode(v)
				return true
			})
			if _, out, e := runDiagnostic(t, s.URL, diagnosticKeyFile(t), "status"); e == nil || out != "" {
				t.Fatal("out-of-contract decoded publication accepted")
			}
		})
	}
}

func TestControllerDiagnosticsPublicationEncodedReadIsBounded(t *testing.T) {
	s := diagnosticServer(t, diagnosticObservation(), func(w http.ResponseWriter, r *http.Request) bool {
		if r.URL.Path != "/api/v1/publication" {
			return false
		}
		w.Header().Set("Content-Type", "application/json")
		chunk := []byte(strings.Repeat(" ", 1<<20))
		for n := 0; n < 50; n++ {
			if _, err := w.Write(chunk); err != nil {
				break
			}
		}
		return true
	})
	if _, out, e := runDiagnostic(t, s.URL, diagnosticKeyFile(t), "status", "--timeout", "30s"); e == nil || out != "" {
		t.Fatal("unbounded encoded publication accepted")
	}
}
func TestControllerDiagnosticsIncompleteReportsFailDoctor(t *testing.T) {
	for _, tc := range []struct {
		name, want string
		change     func(map[string]any)
	}{
		{"disconnected", "disconnected", func(n map[string]any) {
			n["connection"] = "disconnected"
			n["freshness"] = "stale"
			n["fresh_for_millis"] = 0
		}},
		{"stale", "stale", func(n map[string]any) { n["freshness"] = "stale"; n["fresh_for_millis"] = 0 }},
		{"received", "applying", func(n map[string]any) { n["report"].(map[string]any)["phase"] = "RECEIVED" }},
		{"failed", "failed", func(n map[string]any) { n["report"].(map[string]any)["phase"] = "FAILED" }},
		{"different revision", "applying", func(n map[string]any) { n["report"].(map[string]any)["revision"] = "2"; n["matches_desired"] = false }},
		{"different hash", "applying", func(n map[string]any) {
			n["report"].(map[string]any)["sha256"] = strings.Repeat("b", 64)
			n["matches_desired"] = false
		}},
		{"no report", "unknown", func(n map[string]any) {
			n["report"] = nil
			n["freshness"] = "unknown"
			n["fresh_for_millis"] = 0
			n["matches_desired"] = false
		}},
		{"unbound", "unbound", func(n map[string]any) {
			n["desired_revision"] = ""
			n["desired_sha256"] = ""
			n["matches_desired"] = false
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			o := diagnosticObservation()
			tc.change(o["nodes"].([]any)[0].(map[string]any))
			s := diagnosticServer(t, o, nil)
			key := diagnosticKeyFile(t)
			v, _, e := runDiagnostic(t, s.URL, key, "status")
			if e != nil {
				t.Fatal(e)
			}
			if v["nodes"].([]any)[0].(map[string]any)["state"] != tc.want {
				t.Fatal(v)
			}
			if _, out, e := runDiagnostic(t, s.URL, key, "doctor"); e == nil || out == "" {
				t.Fatal("doctor did not report an incomplete state")
			}
		})
	}
}
func TestControllerDiagnosticsEmptyScopeIsNotHealthy(t *testing.T) {
	o := diagnosticObservation()
	o["nodes"] = []any{}
	s := diagnosticServer(t, o, nil)
	key := diagnosticKeyFile(t)
	if _, _, e := runDiagnostic(t, s.URL, key, "status"); e != nil {
		t.Fatal(e)
	}
	if _, out, e := runDiagnostic(t, s.URL, key, "doctor"); e == nil || out == "" {
		t.Fatal("empty doctor passed")
	}
	if _, out, e := runDiagnostic(t, s.URL, key, "status", "--agent", "missing"); e == nil || out != "" {
		t.Fatal("missing credentialed scope accepted")
	}
}
func TestControllerDiagnosticsRejectsIncoherentOrMalformedRead(t *testing.T) {
	for _, kind := range []string{"authority", "desired revision", "duplicate", "unknown", "case alias", "null value", "trailing", "null", "bad id", "bad hash", "too many", "oversized", "denied", "html", "race"} {
		t.Run(kind, func(t *testing.T) {
			o := diagnosticObservation()
			n := o["nodes"].([]any)[0].(map[string]any)
			switch kind {
			case "authority":
				o["authority"] = "other"
			case "desired revision":
				n["desired_revision"] = "2"
			case "bad id":
				n["agent_ref"] = "node\x1b[31m"
			case "bad hash":
				n["desired_sha256"] = "bad"
			case "null":
				o["nodes"] = nil
			case "case alias":
				o["Authority"] = "authority:test"
			case "null value":
				n["report"].(map[string]any)["observed_at"] = nil
			case "too many":
				a := []any{}
				for i := 0; i < 4097; i++ {
					a = append(a, n)
				}
				o["nodes"] = a
			}
			var publications atomic.Int32
			s := diagnosticServer(t, o, func(w http.ResponseWriter, r *http.Request) bool {
				if r.URL.Path == "/api/v1/publication" && kind == "race" && publications.Add(1) > 1 {
					w.Header().Set("Content-Type", "application/json")
					fmt.Fprint(w, `{"authority":"authority:test","revision":2,"policy":"","facts":""}`)
					return true
				}
				if r.URL.Path != "/api/v1/agent-observations" {
					return false
				}
				w.Header().Set("Content-Type", "application/json")
				switch kind {
				case "duplicate":
					fmt.Fprint(w, `{"authority":"a","authority":"b","observed_at":"2026-01-01T00:00:00Z","nodes":[]}`)
				case "unknown":
					o["unexpected"] = "PRIVATE-ERROR"
					_ = json.NewEncoder(w).Encode(o)
				case "trailing":
					_ = json.NewEncoder(w).Encode(o)
					fmt.Fprint(w, `{}`)
				case "oversized":
					fmt.Fprint(w, strings.Repeat(" ", 4<<20+1))
				case "denied":
					w.WriteHeader(403)
					fmt.Fprint(w, "PRIVATE-ERROR "+diagnosticKey)
				case "html":
					w.Header().Set("Content-Type", "text/html")
					fmt.Fprint(w, "PRIVATE-ERROR")
				default:
					return false
				}
				return true
			})
			if _, out, e := runDiagnostic(t, s.URL, diagnosticKeyFile(t), "status"); e == nil || out != "" {
				t.Fatal("invalid state produced report", out, e)
			}
		})
	}
}
func TestControllerDiagnosticsReadinessAndElapsedTTL(t *testing.T) {
	for _, kind := range []string{"unready", "elapsed", "timeout"} {
		t.Run(kind, func(t *testing.T) {
			o := diagnosticObservation()
			if kind == "elapsed" {
				o["nodes"].([]any)[0].(map[string]any)["fresh_for_millis"] = 25
			}
			var publications atomic.Int32
			s := diagnosticServer(t, o, func(w http.ResponseWriter, r *http.Request) bool {
				if kind == "unready" && r.URL.Path == "/readyz" {
					w.WriteHeader(503)
					return true
				}
				if kind == "elapsed" && r.URL.Path == "/api/v1/publication" && publications.Add(1) > 1 {
					time.Sleep(60 * time.Millisecond)
				}
				if kind == "timeout" {
					time.Sleep(80 * time.Millisecond)
				}
				return false
			})
			args := []string{}
			if kind == "timeout" {
				args = []string{"--timeout", "10ms"}
			}
			v, out, e := runDiagnostic(t, s.URL, diagnosticKeyFile(t), "doctor", args...)
			if e == nil {
				t.Fatal("doctor passed", v)
			}
			if kind == "timeout" && out != "" {
				t.Fatal("timeout emitted report")
			}
			if kind == "elapsed" && v["nodes"].([]any)[0].(map[string]any)["state"] != "stale" {
				t.Fatal(v)
			}
		})
	}
}
func TestControllerDiagnosticsRedirectDoesNotSendCredential(t *testing.T) {
	var hits atomic.Int32
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hits.Add(1) }))
	defer target.Close()
	s := diagnosticServer(t, diagnosticObservation(), func(w http.ResponseWriter, r *http.Request) bool {
		http.Redirect(w, r, target.URL+"/PRIVATE-ERROR", 302)
		return true
	})
	if _, out, e := runDiagnostic(t, s.URL, diagnosticKeyFile(t), "status"); e == nil || out != "" || hits.Load() != 0 {
		t.Fatal("redirect followed or reported")
	}
}
func TestControllerDiagnosticsNormalTLS(t *testing.T) {
	source := diagnosticServer(t, diagnosticObservation(), nil)
	tlsServer := httptest.NewUnstartedServer(source.Config.Handler)
	tlsServer.Config.ErrorLog = log.New(&bytes.Buffer{}, "", 0)
	tlsServer.StartTLS()
	defer tlsServer.Close()
	ca := filepath.Join(t.TempDir(), "ca.pem")
	if e := os.WriteFile(ca, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: tlsServer.Certificate().Raw}), 0600); e != nil {
		t.Fatal(e)
	}
	key := diagnosticKeyFile(t)
	if _, out, e := runDiagnostic(t, tlsServer.URL, key, "status"); e == nil || out != "" {
		t.Fatal("unknown CA accepted")
	}
	if _, _, e := runDiagnostic(t, tlsServer.URL, key, "status", "--ca-file", ca); e != nil {
		t.Fatal("trusted normal TLS failed", e)
	}
	if _, out, e := runDiagnostic(t, strings.Replace(tlsServer.URL, "127.0.0.1", "localhost", 1), key, "status", "--ca-file", ca); e == nil || out != "" {
		t.Fatal("wrong hostname accepted")
	}
}
func TestControllerDiagnosticsUnsafeInputsNeverReachServer(t *testing.T) {
	requireDiagnostics(t)
	var hits atomic.Int32
	s := diagnosticServer(t, diagnosticObservation(), func(w http.ResponseWriter, r *http.Request) bool { hits.Add(1); return false })
	key := diagnosticKeyFile(t)
	for _, origin := range []string{"http://example.test", "http://localhost:1234", s.URL + "/path", s.URL + "?private=PRIVATE-ERROR", s.URL + "/#secret", "https://user:PRIVATE-ERROR@example.test"} {
		if _, out, e := runDiagnostic(t, origin, key, "status"); e == nil || out != "" {
			t.Fatal("unsafe origin accepted")
		}
	}
	for _, kind := range []string{"public", "symlink", "directory", "short", "multiline", "relative"} {
		t.Run(kind, func(t *testing.T) {
			p := diagnosticKeyFile(t)
			switch kind {
			case "public":
				_ = os.Chmod(p, 0644)
			case "symlink":
				q := p + ".link"
				_ = os.Symlink(p, q)
				p = q
			case "directory":
				p = filepath.Dir(p)
			case "short":
				_ = os.WriteFile(p, []byte("short"), 0600)
			case "multiline":
				_ = os.WriteFile(p, []byte(diagnosticKey+"\nPRIVATE-ERROR"), 0600)
			case "relative":
				p = "PRIVATE-ERROR"
			}
			if _, out, e := runDiagnostic(t, s.URL, p, "status"); e == nil || out != "" {
				t.Fatal("unsafe key accepted")
			}
		})
	}
	if hits.Load() != 0 {
		t.Fatal("unsafe input sent requests")
	}
	for _, args := range [][]string{{"status"}, {"doctor"}, {"status", "--controller", s.URL, "--key-file", key, "--timeout", "0s"}, {"status", "--controller", s.URL, "--key-file", key, "--timeout", "31s"}, {"status", "--controller", s.URL, "--key-file", key, "--format", "unknown"}} {
		if out, e := execute(args...); e == nil || out != "" {
			t.Fatal("invalid flags produced report")
		}
	}
}
