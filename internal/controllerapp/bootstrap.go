// SPDX-License-Identifier: Apache-2.0
package controllerapp

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/PicketX/PicketX/internal/account"
	"github.com/PicketX/PicketX/internal/managementaccess"
	"github.com/PicketX/PicketX/internal/runtimeconfig"
)

// Prepare inputs before SQLite can commit an initial account. An existing
// database never causes credentials or passwords to be regenerated.
func prepareInitialState(c runtimeconfig.Config) error {
	if !c.InitialSetup {
		return nil
	}
	if _, err := os.Lstat(c.Database); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	dir := filepath.Dir(c.Database)
	if _, err := initialSecret(filepath.Join(dir, "initial-admin-password")); err != nil {
		return err
	}
	recovery, err := initialSecret(filepath.Join(dir, "initial-recovery-key"))
	if err != nil {
		return err
	}
	digest := sha256.Sum256([]byte(recovery))
	data, _ := json.Marshal(map[string]any{"principals": []any{map[string]any{"id": "principal:bootstrap", "sha256": hex.EncodeToString(digest[:]), "roles": []string{"role:admin"}}}})
	inputs := map[string]string{c.Management.Credentials: string(data), c.Credentials: `{"credentials":[]}`}
	if c.Authz.Enabled {
		inputs[c.Authz.Credentials] = `{"callers":[]}`
	}
	for path, content := range inputs {
		if err := initialFile(path, content); err != nil {
			return err
		}
	}
	return nil
}

// Revision zero means no account operation has ever committed. Reuse SaveAccount
// for password hashing, CAS, persistence and audit; never infer bootstrap from
// the number of enabled administrators or presence of a plaintext password file.
func initializeAdmin(c runtimeconfig.Config, a *managementaccess.Access, log *slog.Logger) error {
	if !c.InitialSetup {
		return nil
	}
	page, err := a.Accounts()
	if err != nil {
		return err
	}
	if page.Revision != 0 {
		return nil
	}
	recoveryPath := filepath.Join(filepath.Dir(c.Database), "initial-recovery-key")
	recovery, err := readInitialSecret(recoveryPath)
	if err != nil {
		return err
	}
	principal, err := a.Authenticate(recovery)
	if err != nil {
		return fmt.Errorf("initial recovery key does not match retained management credentials; restore matching private inputs")
	}
	if allowed, err := a.Allowed(principal, "account:write"); err != nil || !allowed {
		return fmt.Errorf("initial recovery key cannot administer accounts")
	}
	path := filepath.Join(filepath.Dir(c.Database), "initial-admin-password")
	secret, err := readInitialSecret(path)
	if err != nil {
		return fmt.Errorf("initial administrator password unavailable; restore the pending private password file: %w", err)
	}
	if _, _, err := a.SaveAccount(0, principal.ID, "", account.Change{Username: "admin", Password: secret, Roles: []string{"role:admin"}, Reason: "Initialize first administrator"}); err != nil {
		return err
	}
	log.Info("initial administrator created", "username", "admin", "password_file", path, "recovery_key_file", recoveryPath)
	return nil
}

func initialFile(path, content string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	info, err := os.Lstat(dir)
	if err != nil || !info.IsDir() || info.Mode().Perm()&0077 != 0 {
		return fmt.Errorf("initial credential parent must be a private directory: %s", dir)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if os.IsExist(err) {
		info, err := os.Lstat(path)
		if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 {
			return fmt.Errorf("initial credential must be a private regular file: %s", path)
		}
		return nil
	}
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err = f.WriteString(content); err != nil {
		return err
	}
	return f.Sync()
}
func initialSecret(path string) (string, error) {
	if err := initialFile(path, rand.Text()+rand.Text()+"\n"); err != nil {
		return "", err
	}
	return readInitialSecret(path)
}
func readInitialSecret(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("cannot read private initial credential")
	}
	defer f.Close()
	info, err := f.Stat()
	link, linkErr := os.Lstat(path)
	if err != nil || linkErr != nil || !link.Mode().IsRegular() || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 {
		return "", fmt.Errorf("initial credential must be private and regular")
	}
	data, err := io.ReadAll(io.LimitReader(f, 257))
	if err != nil || len(data) > 256 {
		return "", fmt.Errorf("invalid initial credential")
	}
	secret := strings.TrimSuffix(string(data), "\n")
	if len(secret) < 32 || len(secret) > 72 {
		return "", fmt.Errorf("invalid initial credential")
	}
	for _, b := range []byte(secret) {
		if b < 33 || b > 126 {
			return "", fmt.Errorf("invalid initial credential")
		}
	}
	return secret, nil
}
