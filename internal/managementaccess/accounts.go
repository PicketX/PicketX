// SPDX-License-Identifier: Apache-2.0
package managementaccess

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/PicketX/PicketX/internal/account"
	"github.com/PicketX/PicketX/internal/password"
	"github.com/PicketX/PicketX/internal/publication"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const accountPrefix = "principal:account-"

// Password-bearing statements must never reach the SQL logger, even on errors.
func quietAccountsDB(db *gorm.DB) *gorm.DB {
	return db.Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)})
}

func bootstrapAccounts(tx *gorm.DB) error {
	for _, statement := range []string{
		"CREATE TABLE IF NOT EXISTS management_identity_authority (id INTEGER PRIMARY KEY CHECK(id=1), version INTEGER NOT NULL CHECK(version=1), mode TEXT NOT NULL CHECK(mode='database'), revision BIGINT NOT NULL CHECK(revision>=0))",
		"CREATE TABLE IF NOT EXISTS management_accounts (id TEXT PRIMARY KEY, username TEXT NOT NULL UNIQUE, roles TEXT NOT NULL, password_hash TEXT NOT NULL, disabled BOOLEAN NOT NULL, version BIGINT NOT NULL CHECK(version>0))",
		"CREATE TABLE IF NOT EXISTS management_account_audit (revision BIGINT PRIMARY KEY CHECK(revision>0), actor TEXT NOT NULL, account_id TEXT NOT NULL, operation TEXT NOT NULL, reason TEXT NOT NULL, at TEXT NOT NULL)",
	} {
		if err := quietAccountsDB(tx).Exec(statement).Error; err != nil {
			return err
		}
	}
	for _, op := range []string{"account:read", "account:write"} {
		for _, rule := range []struct{ role, effect string }{{"role:admin", "allow"}, {"role:blocked", "deny"}} {
			if err := tx.Exec("INSERT INTO casbin_rule(ptype,v0,v1,v2,v3,v4,v5) SELECT 'p',?,?,?,'','','' WHERE NOT EXISTS (SELECT 1 FROM casbin_rule WHERE ptype='p' AND v0=? AND v1=?)", rule.role, op, rule.effect, rule.role, op).Error; err != nil {
				return err
			}
		}
	}
	return tx.Exec("UPDATE management_schema SET version=3 WHERE id=1").Error
}

type identityAuthority struct {
	ID, Version int
	Mode        string
	Revision    int64
}
type storedAccount struct {
	ID, Username, Roles, PasswordHash string
	Disabled                          bool
	Version                           int64
}
type storedAccountEvent struct {
	Revision                                int64
	Actor, AccountID, Operation, Reason, At string
}

func readAuthority(tx *gorm.DB) (*identityAuthority, error) {
	var rows []identityAuthority
	if err := tx.Table("management_identity_authority").Find(&rows).Error; err != nil || len(rows) > 1 {
		return nil, ErrUnavailable
	}
	if len(rows) == 0 {
		return nil, nil
	}
	r := rows[0]
	if r.ID != 1 || r.Version != 1 || r.Mode != "database" || r.Revision < 0 {
		return nil, ErrUnavailable
	}
	return &r, nil
}

// The private file remains the sole API-key source. Database human IDs are a
// reserved namespace, and an unblocked admin key remains the recovery path.
func validateDatabaseKeys(doc credentialDocument) bool {
	if len(doc.Users) != 0 {
		return false
	}
	recovery := false
	for _, key := range *doc.Principals {
		if strings.HasPrefix(key.ID, accountPrefix) {
			return false
		}
		admin, blocked := false, false
		for _, role := range key.Roles {
			admin = admin || role == "role:admin"
			blocked = blocked || role == "role:blocked"
		}
		recovery = recovery || admin && !blocked
	}
	return recovery
}

func (a *Access) initializeAccounts() error {
	doc, err := loadCredentials(a.settings.Credentials)
	if err != nil {
		return ErrUnavailable
	}
	if a.settings.UsersMode == "database" && !validateDatabaseKeys(doc) {
		return ErrUnavailable
	}
	return a.db.Transaction(func(tx *gorm.DB) error {
		authority, err := readAuthority(tx)
		if err != nil {
			return err
		}
		if a.settings.UsersMode == "file" {
			if authority != nil {
				return ErrUnavailable
			}
			return noOrphanAccounts(tx)
		}
		if authority == nil {
			if err = noOrphanAccounts(tx); err != nil {
				return err
			}
			if err = tx.Exec("INSERT INTO management_identity_authority(id,version,mode,revision) VALUES(1,1,'database',0)").Error; err != nil {
				return ErrUnavailable
			}
		}
		_, err = readAccountRows(tx)
		return err
	})
}

func noOrphanAccounts(tx *gorm.DB) error {
	for _, table := range []string{"management_accounts", "management_account_audit"} {
		var count int64
		if err := tx.Table(table).Count(&count).Error; err != nil || count != 0 {
			return ErrUnavailable
		}
	}
	return nil
}

func accountRoles(roles []string) bool {
	if len(roles) < 1 || len(roles) > 4 {
		return false
	}
	seen := map[string]bool{}
	for _, role := range roles {
		if seen[role] {
			return false
		}
		seen[role] = true
		switch role {
		case "role:admin", "role:viewer", "role:requester", "role:blocked":
		default:
			return false
		}
	}
	return true
}
func accountID(id string) bool {
	if len(id) != len(accountPrefix)+32 || !strings.HasPrefix(id, accountPrefix) {
		return false
	}
	bytes, err := hex.DecodeString(strings.TrimPrefix(id, accountPrefix))
	return err == nil && hex.EncodeToString(bytes) == strings.TrimPrefix(id, accountPrefix)
}
func validAccountReason(reason string) bool {
	return len(reason) <= 512 && utf8.ValidString(reason) && strings.TrimSpace(reason) != ""
}

func readAccountRows(tx *gorm.DB) ([]userRecord, error) {
	var rows []storedAccount
	if err := tx.Table("management_accounts").Order("username").Limit(129).Find(&rows).Error; err != nil || len(rows) > 128 {
		return nil, ErrUnavailable
	}
	users := make([]userRecord, 0, len(rows))
	names := map[string]bool{}
	for _, r := range rows {
		var roles []string
		if json.Unmarshal([]byte(r.Roles), &roles) != nil || !accountID(r.ID) || !validUsername(r.Username) || names[r.Username] || !accountRoles(roles) || !password.ValidHash(r.PasswordHash) || r.Version <= 0 {
			return nil, ErrUnavailable
		}
		names[r.Username] = true
		users = append(users, userRecord{ID: r.ID, Username: r.Username, PasswordHash: r.PasswordHash, Roles: roles, Disabled: r.Disabled, Version: uint64(r.Version)})
	}
	return users, nil
}

func (a *Access) credentials() (credentialDocument, error) {
	doc, err := loadCredentials(a.settings.Credentials)
	if err != nil {
		return credentialDocument{}, ErrUnavailable
	}
	err = a.db.Transaction(func(tx *gorm.DB) error {
		authority, err := readAuthority(tx)
		if err != nil {
			return err
		}
		if a.settings.UsersMode == "file" {
			if authority != nil {
				return ErrUnavailable
			}
			return noOrphanAccounts(tx)
		}
		if authority == nil || !validateDatabaseKeys(doc) {
			return ErrUnavailable
		}
		doc.Users, err = readAccountRows(tx)
		for _, u := range doc.Users {
			if u.Version > uint64(authority.Revision) {
				return ErrUnavailable
			}
		}
		return err
	}, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return credentialDocument{}, ErrUnavailable
	}
	return doc, nil
}

func publicAccount(u userRecord) account.Account {
	return account.Account{ID: u.ID, Username: u.Username, Roles: append([]string(nil), u.Roles...), Disabled: u.Disabled, Version: u.Version}
}

func (a *Access) Accounts() (account.Page, error) {
	page := account.Page{Mode: "database", Accounts: []account.Account{}, Events: []account.Event{}}
	if a.settings.UsersMode != "database" {
		return account.Page{}, ErrUnavailable
	}
	doc, err := loadCredentials(a.settings.Credentials)
	if err != nil || !validateDatabaseKeys(doc) {
		return account.Page{}, ErrUnavailable
	}
	err = a.db.Transaction(func(tx *gorm.DB) error {
		authority, err := readAuthority(tx)
		if err != nil || authority == nil {
			return ErrUnavailable
		}
		users, err := readAccountRows(tx)
		if err != nil {
			return err
		}
		page.Revision = uint64(authority.Revision)
		for _, u := range users {
			if u.Version > page.Revision {
				return ErrUnavailable
			}
			page.Accounts = append(page.Accounts, publicAccount(u))
		}
		var events []storedAccountEvent
		if err = tx.Table("management_account_audit").Order("revision DESC").Limit(50).Find(&events).Error; err != nil {
			return ErrUnavailable
		}
		for _, e := range events {
			at, err := time.Parse(time.RFC3339Nano, e.At)
			if err != nil || e.Revision <= 0 || e.Revision > authority.Revision || !identifier(e.Actor, "principal:") || !accountID(e.AccountID) || (e.Operation != "create" && e.Operation != "update") || !validAccountReason(e.Reason) {
				return ErrUnavailable
			}
			page.Events = append(page.Events, account.Event{Actor: e.Actor, AccountID: e.AccountID, Operation: e.Operation, Reason: e.Reason, Revision: uint64(e.Revision), At: at})
		}
		page.ObservedAt = time.Now().UTC()
		return nil
	}, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return account.Page{}, ErrUnavailable
	}
	return page, nil
}

func (a *Access) SaveAccount(expected uint64, actor, id string, change account.Change) (account.Account, uint64, error) {
	if a.settings.UsersMode != "database" {
		return account.Account{}, 0, ErrUnavailable
	}
	create := id == ""
	if !identifier(actor, "principal:") || !accountRoles(change.Roles) || !validAccountReason(change.Reason) || expected >= math.MaxInt64 || create && !validUsername(change.Username) || !create && (!accountID(id) || change.Username != "") || create && !password.Valid(change.Password) || change.Password != "" && !password.Valid(change.Password) {
		return account.Account{}, 0, account.ErrInvalid
	}
	doc, err := loadCredentials(a.settings.Credentials)
	if err != nil || !validateDatabaseKeys(doc) {
		return account.Account{}, 0, ErrUnavailable
	}
	hash := ""
	if change.Password != "" {
		select {
		case a.passwordSlots <- struct{}{}:
			defer func() { <-a.passwordSlots }()
		default:
			return account.Account{}, 0, ErrUnavailable
		}
		hash, err = password.Hash(change.Password)
		if err != nil {
			return account.Account{}, 0, ErrUnavailable
		}
	}
	if create {
		var random [16]byte
		if _, err = rand.Read(random[:]); err != nil {
			return account.Account{}, 0, ErrUnavailable
		}
		id = accountPrefix + hex.EncodeToString(random[:])
	}
	var result account.Account
	err = a.db.Transaction(func(tx *gorm.DB) error {
		// Acquire write serialization before reading accounts. CAS never compares
		// publication revisions and a failed append rolls this increment back.
		r := tx.Exec("UPDATE management_identity_authority SET revision=revision+1 WHERE id=1 AND version=1 AND mode='database' AND revision=?", int64(expected))
		if r.Error != nil {
			return ErrUnavailable
		}
		if r.RowsAffected != 1 {
			return publication.ErrPrecondition
		}
		users, err := readAccountRows(tx)
		if err != nil {
			return err
		}
		next := userRecord{ID: id, Username: change.Username, Roles: append([]string(nil), change.Roles...), Disabled: change.Disabled, PasswordHash: hash, Version: 1}
		operation := "create"
		if create {
			if len(users) >= 128 {
				return account.ErrInvalid
			}
			for _, u := range users {
				if u.Username == change.Username {
					return account.ErrInvalid
				}
			}
		} else {
			found := false
			for _, u := range users {
				if u.ID == id {
					next.Username = u.Username
					next.Version = u.Version + 1
					if hash == "" {
						next.PasswordHash = u.PasswordHash
					}
					found = true
				}
			}
			if !found {
				return account.ErrNotFound
			}
			operation = "update"
		}
		roles, _ := json.Marshal(next.Roles)
		row := storedAccount{ID: next.ID, Username: next.Username, Roles: string(roles), PasswordHash: next.PasswordHash, Disabled: next.Disabled, Version: int64(next.Version)}
		if create {
			err = tx.Table("management_accounts").Create(&row).Error
		} else {
			err = tx.Table("management_accounts").Where("id=?", id).Select("roles", "password_hash", "disabled", "version").Updates(&row).Error
		}
		if err != nil {
			return ErrUnavailable
		}
		event := storedAccountEvent{Revision: int64(expected + 1), Actor: actor, AccountID: id, Operation: operation, Reason: change.Reason, At: time.Now().UTC().Format(time.RFC3339Nano)}
		if err = tx.Table("management_account_audit").Create(&event).Error; err != nil {
			return ErrUnavailable
		}
		result = publicAccount(next)
		return nil
	})
	if err != nil {
		if errors.Is(err, publication.ErrPrecondition) || errors.Is(err, account.ErrInvalid) || errors.Is(err, account.ErrNotFound) {
			return account.Account{}, 0, err
		}
		return account.Account{}, 0, ErrUnavailable
	}
	return result, expected + 1, nil
}
