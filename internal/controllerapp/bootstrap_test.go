// SPDX-License-Identifier: Apache-2.0
package controllerapp

import (
	"bytes"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PicketX/PicketX/internal/account"
	"github.com/PicketX/PicketX/internal/managementaccess"
	"github.com/PicketX/PicketX/internal/publication"
	"github.com/PicketX/PicketX/internal/runtimeconfig"
)

func initialConfig(t *testing.T) runtimeconfig.Config {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "controller")
	c, err := runtimeconfig.LoadDefault(nil, map[string]string{"database": filepath.Join(dir, "controller.sqlite"), "credentials": filepath.Join(dir, "agents.json"), "management.credentials": filepath.Join(dir, "management.json"), "authz.credentials": filepath.Join(dir, "callers.json")})
	if err != nil {
		t.Fatal(err)
	}
	return c
}
func openInitialAccounts(t *testing.T, c runtimeconfig.Config) *managementaccess.Access {
	t.Helper()
	store, err := publication.OpenMode(c.Database, c.Agent.Authority, c.Source.Mode)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	a, err := managementaccess.New(store.GORM(), managementaccess.Settings{Credentials: c.Management.Credentials, UsersMode: c.Management.UsersMode, Reload: c.Management.Reload, WarnRules: c.Management.WarnRules, WarnLoadPercent: c.Management.WarnLoadPercent}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	return a
}
func TestInitialAdminCreatedOnceAndNeverReactivated(t *testing.T) {
	c := initialConfig(t)
	if err := prepareInitialState(c); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(filepath.Dir(c.Database), "initial-admin-password")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	secret := strings.TrimSpace(string(raw))
	a := openInitialAccounts(t, c)
	var logs bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&logs, nil))
	if err := initializeAdmin(c, a, log); err != nil {
		t.Fatal(err)
	}
	p, _, err := a.AuthenticatePassword("admin", secret)
	if err != nil {
		t.Fatal("initial password rejected", err)
	}
	if len(p.Roles) != 1 || p.Roles[0] != "role:admin" {
		t.Fatal("missing admin role")
	}
	page, err := a.Accounts()
	if err != nil || page.Revision != 1 || len(page.Accounts) != 1 || len(page.Events) != 1 {
		t.Fatal("missing durable account/audit", err)
	}
	for _, file := range []string{path, c.Management.Credentials, c.Credentials, c.Authz.Credentials, filepath.Join(filepath.Dir(c.Database), "initial-recovery-key")} {
		info, err := os.Lstat(file)
		if err != nil || info.Mode().Perm() != 0600 {
			t.Fatal("nonprivate bootstrap file", file, err)
		}
	}
	if strings.Contains(logs.String(), secret) {
		t.Fatal("password in logs")
	}
	if err := prepareInitialState(c); err != nil {
		t.Fatal(err)
	}
	if err := initializeAdmin(c, a, log); err != nil {
		t.Fatal(err)
	}
	after, _ := a.Accounts()
	if after.Revision != 1 {
		t.Fatal("bootstrap repeated")
	}
	// Deliberate admin changes, including disabling it, must survive any restart/bootstrap attempt.
	if _, _, err := a.SaveAccount(1, p.ID, page.Accounts[0].ID, account.Change{Roles: []string{"role:admin"}, Disabled: true, Password: "changed-random-password", Reason: "Disable initial admin"}); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := prepareInitialState(c); err != nil {
		t.Fatal(err)
	}
	if err := initializeAdmin(c, a, log); err != nil {
		t.Fatal(err)
	}
	after, _ = a.Accounts()
	if after.Revision != 2 || len(after.Accounts) != 1 || !after.Accounts[0].Disabled {
		t.Fatal("admin recreated")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("password regenerated")
	}
	if err := os.Remove(c.Credentials); err != nil {
		t.Fatal(err)
	}
	if err := prepareInitialState(c); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(c.Credentials); !os.IsNotExist(err) {
		t.Fatal("lost credential file regenerated for existing DB")
	}
}
func TestInitialAdminResumesPendingSetupAndRejectsMissingSecret(t *testing.T) {
	c := initialConfig(t)
	if err := prepareInitialState(c); err != nil {
		t.Fatal(err)
	}
	a := openInitialAccounts(t, c) // Simulate interruption after migration, before account creation.
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	if err := prepareInitialState(c); err != nil {
		t.Fatal(err)
	}
	if err := initializeAdmin(c, a, log); err != nil {
		t.Fatal("pending setup did not resume", err)
	}
	other := initialConfig(t)
	if err := prepareInitialState(other); err != nil {
		t.Fatal(err)
	}
	empty := openInitialAccounts(t, other)
	os.Remove(filepath.Join(filepath.Dir(other.Database), "initial-admin-password"))
	if err := initializeAdmin(other, empty, log); err == nil {
		t.Fatal("missing pending password accepted")
	}
}
func TestInitialStateRejectsUnsafeSecretAndPreservesFiles(t *testing.T) {
	c := initialConfig(t)
	if err := prepareInitialState(c); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(filepath.Dir(c.Database), "initial-admin-password")
	before, _ := os.ReadFile(path)
	if err := prepareInitialState(c); err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(before, after) {
		t.Fatal("pending password replaced")
	}
	os.Chmod(path, 0644)
	if err := prepareInitialState(c); err == nil {
		t.Fatal("unsafe password permissions accepted")
	}
	os.Remove(path)
	target := filepath.Join(t.TempDir(), "secret")
	os.WriteFile(target, []byte("untouched"), 0600)
	os.Symlink(target, path)
	if err := prepareInitialState(c); err == nil {
		t.Fatal("password symlink accepted")
	}
	data, _ := os.ReadFile(target)
	if string(data) != "untouched" {
		t.Fatal("symlink target altered")
	}
}

func TestInitialAdminRejectsUnmatchedRecoveryKey(t *testing.T) {
	original := initialConfig(t)
	if err := prepareInitialState(original); err != nil {
		t.Fatal(err)
	}
	preserved, err := os.ReadFile(original.Management.Credentials)
	if err != nil {
		t.Fatal(err)
	}
	c := initialConfig(t)
	if err := os.MkdirAll(filepath.Dir(c.Management.Credentials), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(c.Management.Credentials, preserved, 0600); err != nil {
		t.Fatal(err)
	}
	if err := prepareInitialState(c); err != nil {
		t.Fatal(err)
	}
	a := openInitialAccounts(t, c)
	if err := initializeAdmin(c, a, slog.New(slog.NewTextHandler(io.Discard, nil))); err == nil {
		t.Fatal("bootstrap accepted an unrelated recovery key")
	}
	page, err := a.Accounts()
	if err != nil || page.Revision != 0 || len(page.Accounts) != 0 {
		t.Fatal("failed bootstrap changed accounts", err)
	}
	after, _ := os.ReadFile(c.Management.Credentials)
	if !bytes.Equal(after, preserved) {
		t.Fatal("existing management credentials replaced")
	}
}
