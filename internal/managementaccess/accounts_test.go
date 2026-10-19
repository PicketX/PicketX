// SPDX-License-Identifier: Apache-2.0
package managementaccess

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/PicketX/PicketX/internal/account"
	"github.com/PicketX/PicketX/internal/management"
	"github.com/PicketX/PicketX/internal/publication"
)

func databaseAccess(t *testing.T) (*Access, *publication.Store, string) {
	t.Helper()
	_, store, path := fixture(t)
	a, err := New(store.GORM(), Settings{Credentials: path, UsersMode: "database", Reload: time.Second, WarnRules: 10000, WarnLoadPercent: 50}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	return a, store, path
}

func TestDatabaseAccountsConcurrentUpdateHasOneWinner(t *testing.T) {
	a, _, _ := databaseAccess(t)
	created, _, err := a.SaveAccount(0, "principal:operator", "", account.Change{Username: "alice", Password: "a real human password!", Roles: []string{"role:requester"}, Reason: "seed"})
	if err != nil {
		t.Fatal(err)
	}
	gate := make(chan struct{})
	results := make(chan error, 2)
	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-gate
			_, _, err := a.SaveAccount(1, "principal:operator", created.ID, account.Change{Roles: []string{"role:viewer"}, Reason: "concurrent update"})
			results <- err
		}()
	}
	close(gate)
	wg.Wait()
	close(results)
	winners, stale := 0, 0
	for err := range results {
		if err == nil {
			winners++
		} else if errors.Is(err, publication.ErrPrecondition) {
			stale++
		} else {
			t.Fatal(err)
		}
	}
	p, err := a.Accounts()
	if err != nil || winners != 1 || stale != 1 || p.Revision != 2 || len(p.Events) != 2 || p.Accounts[0].Version != 2 {
		t.Fatal(winners, stale, p, err)
	}
}

func TestDatabaseAccountsStickyAuthorityAndRecoveryKeys(t *testing.T) {
	file, store, path := fixture(t)
	settings := file.settings
	settings.UsersMode = "database"
	writeCredentials(t, path, `["role:admin","role:blocked"]`)
	if _, err := New(store.GORM(), settings, file.log); err == nil {
		t.Fatal("blocked bootstrap admin")
	}
	writeCredentials(t, path, `["role:viewer"]`)
	if _, err := New(store.GORM(), settings, file.log); err == nil {
		t.Fatal("no admin recovery key")
	}
	writeCredentials(t, path, `["role:admin"]`)
	a, err := New(store.GORM(), settings, file.log)
	if err != nil {
		t.Fatal(err)
	}
	if file.Ready() {
		t.Fatal("existing file instance remained ready after authority change")
	}
	if _, err = file.Authenticate(token); !errors.Is(err, ErrUnavailable) {
		t.Fatal("file instance used stale authority", err)
	}
	if !a.Ready() {
		t.Fatal("DB authority not ready")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(path, []byte(strings.Replace(string(raw), "principal:operator", "principal:account-00000000000000000000000000000000", 1)), 0600); err != nil {
		t.Fatal(err)
	}
	if a.Ready() {
		t.Fatal("key entered human namespace")
	}
}

func TestDatabaseAccountResetAndDuplicateDoNotLeakOrAdvanceOnError(t *testing.T) {
	a, store, _ := databaseAccess(t)
	input := account.Change{Username: "alice", Password: "a real human password!", Roles: []string{"role:requester"}, Reason: "seed"}
	created, _, err := a.SaveAccount(0, "principal:operator", "", input)
	if err != nil {
		t.Fatal(err)
	}
	_, binding, err := a.AuthenticatePassword(input.Username, input.Password)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = a.SaveAccount(1, "principal:operator", "", input); !errors.Is(err, account.ErrInvalid) {
		t.Fatal("duplicate username", err)
	}
	input.Username = ""
	input.Password = "a new real human password!"
	input.Reason = "reset"
	if _, _, err = a.SaveAccount(1, "principal:operator", created.ID, input); err != nil {
		t.Fatal(err)
	}
	if _, err = a.ResolveSession(created.ID, binding); !errors.Is(err, ErrUnauthorized) {
		t.Fatal("reset retained session", err)
	}
	if _, _, err = a.AuthenticatePassword("alice", "a real human password!"); !errors.Is(err, ErrUnauthorized) {
		t.Fatal("old password accepted", err)
	}
	if _, _, err = a.AuthenticatePassword("alice", input.Password); err != nil {
		t.Fatal(err)
	}
	if err = store.GORM().Exec("UPDATE management_accounts SET roles='[\"role:custom\"]'").Error; err != nil {
		t.Fatal(err)
	}
	if a.Ready() {
		t.Fatal("corrupt account roles ready")
	}
	if _, err = a.Accounts(); !errors.Is(err, ErrUnavailable) {
		t.Fatal("corrupt account read", err)
	}
}
func TestDatabaseAccountLifecycleCASAuditAndPermanentSessionRevocation(t *testing.T) {
	a, store, path := databaseAccess(t)
	p, err := a.Accounts()
	if err != nil || p.Revision != 0 || len(p.Accounts) != 0 {
		t.Fatal(p, err)
	}
	input := account.Change{Username: "alice", Password: "a real human password!", Roles: []string{"role:requester"}, Reason: "create human"}
	created, rev, err := a.SaveAccount(0, "principal:operator", "", input)
	if err != nil || rev != 1 || created.Version != 1 || !strings.HasPrefix(created.ID, "principal:account-") {
		t.Fatal(created, rev, err)
	}
	principal, binding, err := a.AuthenticatePassword("alice", input.Password)
	if err != nil || principal.ID != created.ID {
		t.Fatal(principal, err)
	}
	if _, _, err = a.SaveAccount(0, "principal:operator", created.ID, account.Change{Roles: []string{"role:viewer"}, Reason: "stale"}); !errors.Is(err, publication.ErrPrecondition) {
		t.Fatal("stale update", err)
	}
	change := account.Change{Roles: []string{"role:viewer"}, Disabled: true, Reason: "disable"}
	disabled, rev, err := a.SaveAccount(1, "principal:operator", created.ID, change)
	if err != nil || rev != 2 || disabled.Version != 2 {
		t.Fatal(disabled, rev, err)
	}
	change.Disabled = false
	change.Reason = "enable"
	enabled, rev, err := a.SaveAccount(2, "principal:operator", created.ID, change)
	if err != nil || rev != 3 || enabled.Version != 3 {
		t.Fatal(enabled, rev, err)
	}
	if _, err = a.ResolveSession(principal.ID, binding); !errors.Is(err, ErrUnauthorized) {
		t.Fatal("old cookie resurrected", err)
	}
	fresh, newBinding, err := a.AuthenticatePassword("alice", input.Password)
	if err != nil || fresh.Roles[0] != "role:viewer" || newBinding == binding {
		t.Fatal(fresh, err)
	}
	p, err = a.Accounts()
	if err != nil || p.Revision != 3 || len(p.Events) != 3 || p.Events[0].Actor != "principal:operator" {
		t.Fatal(p, err)
	}
	raw, _ := json.Marshal(p)
	if strings.Contains(string(raw), input.Password) || strings.Contains(string(raw), "password_hash") || strings.Contains(string(raw), "$2a$") {
		t.Fatal("credential leak")
	}
	reopened, err := New(store.GORM(), a.settings, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = reopened.ResolveSession(principal.ID, binding); !errors.Is(err, ErrUnauthorized) {
		t.Fatal("old binding accepted on reopen", err)
	}
	if _, err = reopened.ResolveSession(fresh.ID, newBinding); err != nil {
		t.Fatal(err)
	}
	file := a.settings
	file.UsersMode = "file"
	if _, err = New(store.GORM(), file, slog.New(slog.NewTextHandler(io.Discard, nil))); err == nil {
		t.Fatal("implicit file fallback")
	}
	bytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	bytes = []byte(strings.Replace(string(bytes), `"principals":`, `"users":[{"id":"principal:shadow","username":"shadow","password_hash":"`+dummyPasswordHash+`","roles":["role:admin"]}],"principals":`, 1))
	if err = os.WriteFile(path, bytes, 0600); err != nil {
		t.Fatal(err)
	}
	if a.Ready() {
		t.Fatal("mixed user authorities ready")
	}
	if _, _, err = a.AuthenticatePassword("alice", input.Password); !errors.Is(err, ErrUnavailable) {
		t.Fatal("mixed authority login", err)
	}
}
func TestDatabaseAccountAuditFailureRollsBackAndDeniedRolesCannotWrite(t *testing.T) {
	a, store, _ := databaseAccess(t)
	principal, _ := a.Authenticate(token)
	s := management.New(a, nil)
	input := account.Change{Username: "bob", Password: "another human password!", Roles: []string{"role:requester"}, Reason: "create"}
	if _, _, err := s.SaveAccount(Principal{ID: "principal:viewer", Roles: []string{"role:viewer"}}, 0, "", input); !errors.Is(err, management.ErrForbidden) {
		t.Fatal(err)
	}
	if _, _, err := s.SaveAccount(Principal{ID: "principal:blocked", Roles: []string{"role:admin", "role:blocked"}}, 0, "", input); !errors.Is(err, management.ErrForbidden) {
		t.Fatal(err)
	}
	if err := store.GORM().Exec("CREATE TRIGGER fail_account_audit BEFORE INSERT ON management_account_audit BEGIN SELECT RAISE(ABORT,'injected'); END").Error; err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.SaveAccount(principal, 0, "", input); err == nil {
		t.Fatal("audit failure accepted")
	}
	p, err := a.Accounts()
	if err != nil || p.Revision != 0 || len(p.Accounts) != 0 || len(p.Events) != 0 {
		t.Fatal(p, err)
	}
	if err := store.GORM().Exec("DROP TRIGGER fail_account_audit").Error; err != nil {
		t.Fatal(err)
	}
	for _, bad := range []account.Change{{Username: "bad", Password: "short", Roles: input.Roles, Reason: "x"}, {Username: "bad", Password: input.Password, Roles: []string{"role:custom"}, Reason: "x"}, {Username: "bad", Password: input.Password, Roles: input.Roles, Reason: " "}} {
		if _, _, err := s.SaveAccount(principal, 0, "", bad); !errors.Is(err, account.ErrInvalid) {
			t.Fatal("invalid accepted", err)
		}
	}
	if _, _, err := s.SaveAccount(principal, 0, "", input); err != nil {
		t.Fatal(err)
	}
	p, err = a.Accounts()
	if err != nil || p.Revision != 1 || len(p.Events) != 1 {
		t.Fatal(p, err)
	}
}
