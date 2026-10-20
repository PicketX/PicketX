// SPDX-License-Identifier: Apache-2.0
package oidclogin

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/coreos/go-oidc/v3/oidc"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// Real HTTP discovery/token/JWKS plus real RSA signatures, not a verifier stub.
type issuerFixture struct {
	server           *httptest.Server
	key              *rsa.PrivateKey
	exchanges        atomic.Int32
	claims           map[string]any
	nonce, challenge string
	fail             bool
	badSignature     bool
}

func privateJSON(t *testing.T, p string, v any) {
	t.Helper()
	b, _ := json.Marshal(v)
	if e := os.WriteFile(p, b, 0600); e != nil {
		t.Fatal(e)
	}
}
func fixture(t *testing.T) (*Client, *issuerFixture, string) {
	t.Helper()
	key, e := rsa.GenerateKey(rand.Reader, 2048)
	if e != nil {
		t.Fatal(e)
	}
	f := &issuerFixture{key: key}
	f.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			json.NewEncoder(w).Encode(map[string]any{"issuer": f.server.URL, "authorization_endpoint": f.server.URL + "/authorize", "token_endpoint": f.server.URL + "/token", "jwks_uri": f.server.URL + "/keys", "id_token_signing_alg_values_supported": []string{"RS256"}})
		case "/keys":
			json.NewEncoder(w).Encode(map[string]any{"keys": []any{map[string]any{"kty": "RSA", "alg": "RS256", "use": "sig", "kid": "one", "n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()), "e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes())}}})
		case "/token":
			f.exchanges.Add(1)
			r.ParseForm()
			sum := sha256.Sum256([]byte(r.Form.Get("code_verifier")))
			id, secret, _ := r.BasicAuth()
			if id != "client" || secret != "private-client-secret" || r.Form.Get("code") != "one-use-code" || r.Form.Get("redirect_uri") != "http://127.0.0.1:8080/console/api/v1/oidc/callback" || base64.RawURLEncoding.EncodeToString(sum[:]) != f.challenge {
				http.Error(w, "private token failure", 400)
				return
			}
			if f.fail {
				http.Error(w, "private provider diagnostic", 500)
				return
			}
			claims := map[string]any{"iss": f.server.URL, "aud": "client", "sub": "oidc-alice", "nonce": f.nonce, "iat": time.Now().Unix(), "exp": time.Now().Add(time.Hour).Unix()}
			for k, v := range f.claims {
				claims[k] = v
			}
			b, _ := json.Marshal(claims)
			header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","kid":"one"}`))
			input := header + "." + base64.RawURLEncoding.EncodeToString(b)
			hash := sha256.Sum256([]byte(input))
			sig, _ := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, hash[:])
			if f.badSignature {
				sig[0] ^= 1
			}
			json.NewEncoder(w).Encode(map[string]any{"access_token": "not-retained", "token_type": "Bearer", "id_token": input + "." + base64.RawURLEncoding.EncodeToString(sig)})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(f.server.Close)
	dir := t.TempDir()
	bindings := filepath.Join(dir, "bindings.json")
	privateJSON(t, bindings, map[string]any{"bindings": []any{map[string]any{"subject": "oidc-alice", "principal_id": "principal:alice", "generation": strings.Repeat("a", 32)}}})
	config := filepath.Join(dir, "provider.json")
	privateJSON(t, config, map[string]any{"issuer": f.server.URL, "client_id": "client", "client_secret": "private-client-secret", "bindings": bindings})
	c, e := New(context.Background(), config, "http://127.0.0.1:8080")
	if e != nil {
		t.Fatal(e)
	}
	return c, f, bindings
}
func begin(t *testing.T, c *Client, f *issuerFixture) (string, string) {
	t.Helper()
	auth, state, cookie, e := c.Start()
	if e != nil {
		t.Fatal(e)
	}
	u, e := url.Parse(auth)
	if e != nil {
		t.Fatal(e)
	}
	q := u.Query()
	f.nonce = q.Get("nonce")
	f.challenge = q.Get("code_challenge")
	if q.Get("state") != state || len(state) != 64 || len(cookie) != 64 || f.nonce == state || cookie == state || q.Get("code_challenge_method") != "S256" || q.Get("response_type") != "code" || q.Get("scope") != "openid" {
		t.Fatal("unsafe auth request", q)
	}
	return state, cookie
}
func TestOIDCVerifiedProofFreshBindingsAndSingleUse(t *testing.T) {
	c, f, path := fixture(t)
	state, cookie := begin(t, c, f)
	id, e := c.Complete(context.Background(), state, cookie, "one-use-code")
	if e != nil || id.PrincipalID != "principal:alice" || id.Subject != "oidc-alice" || !id.ExpiresAt.After(time.Now()) {
		t.Fatal(id, e)
	}
	if _, e = c.Complete(context.Background(), state, cookie, "one-use-code"); e == nil || f.exchanges.Load() != 1 {
		t.Fatal("replay exchanged", e)
	}
	if p, _, e := c.Resolve(id.Subject, id.Fingerprint); e != nil || p != "principal:alice" {
		t.Fatal(p, e)
	}
	for _, replacement := range []any{map[string]any{"bindings": []any{}}, map[string]any{"bindings": []any{map[string]any{"subject": "oidc-alice", "principal_id": "principal:bob", "generation": strings.Repeat("a", 32)}}}, map[string]any{"bindings": []any{map[string]any{"subject": "oidc-alice", "principal_id": "principal:alice", "generation": strings.Repeat("b", 32)}}}} {
		privateJSON(t, path, replacement)
		if _, _, e := c.Resolve(id.Subject, id.Fingerprint); e == nil {
			t.Fatal("old binding revived")
		}
	}
}
func TestOIDCRejectsHostileClaims(t *testing.T) {
	cases := []map[string]any{{"nonce": "wrong"}, {"iss": "https://evil.example"}, {"aud": "another-client"}, {"exp": time.Now().Add(-time.Hour).Unix()}, {"iat": time.Now().Add(time.Hour).Unix()}, {"sub": ""}, {"sub": "x\n"}, {"aud": []string{"client", "other"}}, {"azp": "other"}}
	for i, claims := range cases {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			c, f, _ := fixture(t)
			f.claims = claims
			state, cookie := begin(t, c, f)
			if _, e := c.Complete(context.Background(), state, cookie, "one-use-code"); e == nil {
				t.Fatal("hostile identity accepted", claims)
			}
		})
	}
}
func TestOIDCBrowserBindingExpiryLimitsAndConcurrentReplay(t *testing.T) {
	c, f, _ := fixture(t)
	state, cookie := begin(t, c, f)
	if _, e := c.Complete(context.Background(), state, "wrong", "one-use-code"); e == nil || f.exchanges.Load() != 0 {
		t.Fatal("cross browser accepted")
	}
	state, cookie = begin(t, c, f)
	var wg sync.WaitGroup
	var ok atomic.Int32
	for i := 0; i < 8; i++ {
		wg.Go(func() {
			if _, e := c.Complete(context.Background(), state, cookie, "one-use-code"); e == nil {
				ok.Add(1)
			}
		})
	}
	wg.Wait()
	if ok.Load() != 1 || f.exchanges.Load() != 1 {
		t.Fatal("concurrent replay", ok.Load(), f.exchanges.Load())
	}
	c.now = func() time.Time { return time.Now().Add(-6 * time.Minute) }
	old, oldCookie := begin(t, c, f)
	c.now = time.Now
	if _, e := c.Complete(context.Background(), old, oldCookie, "one-use-code"); e == nil {
		t.Fatal("expired transaction accepted")
	}
	for i := 0; i < 128; i++ {
		if _, _, _, e := c.Start(); e != nil {
			t.Fatal(i, e)
		}
	}
	if _, _, _, e := c.Start(); e == nil {
		t.Fatal("unbounded pending transactions")
	}
}
func TestOIDCPrivateBindingsFailClosedAndNoProviderDiagnostic(t *testing.T) {
	c, f, path := fixture(t)
	f.fail = true
	state, cookie := begin(t, c, f)
	if _, e := c.Complete(context.Background(), state, cookie, "one-use-code"); e == nil || strings.Contains(e.Error(), "private") {
		t.Fatal("diagnostic escaped", e)
	}
	for _, raw := range []string{`{"bindings":null}`, `{"bindings":[],"oops":1}`, `{"bindings":[],"bindings":[]}`, `{"bindings":[{"subject":"oidc-alice","principal_id":"principal:alice","generation":"bad"}]}`} {
		os.WriteFile(path, []byte(raw), 0600)
		if _, _, e := c.Resolve("oidc-alice", ""); e == nil {
			t.Fatal("invalid binding accepted", raw)
		}
	}
	os.Chmod(path, 0644)
	if _, _, e := c.Resolve("oidc-alice", ""); e == nil {
		t.Fatal("public bindings accepted")
	}
}

func TestOIDCRequiresExactIssuerEvenForLibraryGoogleAlias(t *testing.T) {
	c, f, _ := fixture(t)
	// go-oidc intentionally accepts Google's legacy bare issuer alias. PicketX's
	// explicit issuer binding must not inherit that exception.
	c.issuer = "https://accounts.google.com"
	c.verifier = oidc.NewVerifier(c.issuer, &oidc.StaticKeySet{PublicKeys: []crypto.PublicKey{&f.key.PublicKey}}, &oidc.Config{ClientID: "client"})
	f.claims = map[string]any{"iss": "accounts.google.com"}
	state, cookie := begin(t, c, f)
	if _, e := c.Complete(context.Background(), state, cookie, "one-use-code"); e == nil {
		t.Fatal("non-exact issuer alias accepted")
	}
}

func TestOIDCRejectsBadSignatureAndAcceptsBoundAuthorizedParty(t *testing.T) {
	c, f, _ := fixture(t)
	f.badSignature = true
	state, cookie := begin(t, c, f)
	if _, e := c.Complete(context.Background(), state, cookie, "one-use-code"); e == nil {
		t.Fatal("bad signature accepted")
	}
	f.badSignature = false
	f.claims = map[string]any{"aud": []string{"client", "other"}, "azp": "client"}
	state, cookie = begin(t, c, f)
	if _, e := c.Complete(context.Background(), state, cookie, "one-use-code"); e != nil {
		t.Fatal("valid authorized party rejected", e)
	}
}

func TestOIDCRejectsMalformedPrivateProviderAndUnsafeEndpoints(t *testing.T) {
	_, _, bindings := fixture(t)
	path := filepath.Join(filepath.Dir(bindings), "provider.json")
	original, e := os.ReadFile(path)
	if e != nil {
		t.Fatal(e)
	}
	var cfg map[string]any
	json.Unmarshal(original, &cfg)
	for _, change := range []map[string]any{{"issuer": "http://remote.example"}, {"issuer": "http://localhost:1234"}, {"issuer": "https://user:password@example.test"}, {"issuer": "https://example.test/#fragment"}, {"client_id": ""}, {"client_secret": ""}, {"bindings": "relative.json"}, {"extra": "secret"}} {
		candidate := map[string]any{}
		for k, v := range cfg {
			candidate[k] = v
		}
		for k, v := range change {
			candidate[k] = v
		}
		privateJSON(t, path, candidate)
		if _, e := New(context.Background(), path, "http://127.0.0.1:8080"); e == nil || strings.Contains(e.Error(), "secret") {
			t.Fatal("unsafe provider accepted or diagnostic leak", change, e)
		}
	}
	privateJSON(t, path, cfg)
	os.Chmod(path, 0644)
	if _, e := New(context.Background(), path, "http://127.0.0.1:8080"); e == nil {
		t.Fatal("public client secret accepted")
	}
	os.Chmod(path, 0600)
	for _, target := range []string{"authorization_endpoint", "token_endpoint", "jwks_uri"} {
		t.Run(target, func(t *testing.T) {
			var server *httptest.Server
			server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				v := map[string]any{"issuer": server.URL, "authorization_endpoint": server.URL + "/authorize", "token_endpoint": server.URL + "/token", "jwks_uri": server.URL + "/keys"}
				v[target] = "http://remote.example/private"
				json.NewEncoder(w).Encode(v)
			}))
			defer server.Close()
			candidate := map[string]any{}
			for k, v := range cfg {
				candidate[k] = v
			}
			candidate["issuer"] = server.URL
			privateJSON(t, path, candidate)
			if _, e := New(context.Background(), path, "http://127.0.0.1:8080"); e == nil {
				t.Fatal("unsafe discovered endpoint accepted")
			}
		})
	}
}
