// SPDX-License-Identifier: Apache-2.0
// Package managementaccess resolves trusted API principals and shared role policy.
package managementaccess

import (
	"context"
	"fmt"
	"github.com/PicketX/PicketX/internal/management"
	"log/slog"
	"sync"
	"time"

	"github.com/casbin/casbin/v3"
	"github.com/casbin/casbin/v3/model"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	"gorm.io/gorm"
)

var ErrUnauthorized = management.ErrUnauthorized
var ErrUnavailable = management.ErrUnavailable

type Settings struct {
	Credentials                string
	Reload                     time.Duration
	WarnRules, WarnLoadPercent int
}
type Principal = management.Principal
type Access struct {
	settings  Settings
	enforcer  *casbin.SyncedEnforcer
	log       *slog.Logger
	mu        sync.RWMutex
	available bool
	loadedAt  time.Time
}

const policyModel = `[request_definition]
r = subjects, operation
[policy_definition]
p = sub, operation, eft
[role_definition]
g = _, _
[policy_effect]
e = some(where (p.eft == allow)) && !some(where (p.eft == deny))
[matchers]
m = hasSubject(r.subjects, p.sub) && r.operation == p.operation
`

func New(db *gorm.DB, settings Settings, log *slog.Logger) (*Access, error) {
	if db == nil || log == nil || settings.Credentials == "" || settings.Reload <= 0 || settings.Reload > 24*time.Hour || settings.WarnRules < 1 || settings.WarnLoadPercent < 1 || settings.WarnLoadPercent > 1000 {
		return nil, fmt.Errorf("invalid management access settings")
	}
	if _, e := loadPrincipals(settings.Credentials); e != nil {
		return nil, e
	}
	// Keep the official adapter's own default AutoMigrate separate from business migrations.
	adapter, e := gormadapter.NewAdapterByDB(db)
	if e != nil {
		return nil, e
	}
	if e = bootstrap(db); e != nil {
		return nil, e
	}
	m, e := model.NewModelFromString(policyModel)
	if e != nil {
		return nil, e
	}
	enforcer, e := casbin.NewSyncedEnforcer(m)
	if e != nil {
		return nil, e
	}
	enforcer.SetAdapter(adapter)
	enforcer.AddFunction("hasSubject", func(args ...interface{}) (interface{}, error) {
		if len(args) != 2 {
			return false, fmt.Errorf("invalid subject context")
		}
		subjects, ok := args[0].([]string)
		target, ok2 := args[1].(string)
		if !ok || !ok2 {
			return false, fmt.Errorf("invalid subject context")
		}
		for _, subject := range subjects {
			if subject == target {
				return true, nil
			}
		}
		return false, nil
	})
	a := &Access{settings: settings, enforcer: enforcer, log: log.With("component", "management_rbac")}
	if e = a.reload(true); e != nil {
		return nil, e
	}
	return a, nil
}

func bootstrap(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if e := tx.Exec("CREATE TABLE IF NOT EXISTS management_schema (id INTEGER PRIMARY KEY CHECK(id=1), version INTEGER NOT NULL)").Error; e != nil {
			return e
		}
		var row struct{ Version int }
		r := tx.Raw("SELECT version FROM management_schema WHERE id=1").Scan(&row)
		if r.Error != nil {
			return r.Error
		}
		if r.RowsAffected > 0 {
			if row.Version != 1 {
				return fmt.Errorf("unsupported management schema version")
			}
			return nil
		}
		var count int64
		if e := tx.Table("casbin_rule").Count(&count).Error; e != nil {
			return e
		}
		if count == 0 {
			for _, op := range []string{"publication:read", "publication:write", "audit:read"} {
				for _, role := range []string{"role:admin", "role:viewer", "role:blocked"} {
					if role == "role:viewer" && op == "publication:write" {
						continue
					}
					effect := "allow"
					if role == "role:blocked" {
						effect = "deny"
					}
					if e := tx.Exec("INSERT INTO casbin_rule(ptype,v0,v1,v2,v3,v4,v5) VALUES('p',?,?,?,'','','')", role, op, effect).Error; e != nil {
						return e
					}
				}
			}
		}
		return tx.Exec("INSERT INTO management_schema(id,version) VALUES(1,1)").Error
	})
}

// Allowed performs one combined effect evaluation, after resolving shared role inheritance.
// No user membership is registered in the process-wide enforcer.
func (a *Access) Allowed(p Principal, operation string) (bool, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if !a.available {
		return false, ErrUnavailable
	}
	if p.ID == "" || len(p.Roles) == 0 {
		return false, nil
	}
	subjects := append([]string(nil), p.Roles...)
	for _, role := range p.Roles {
		inherited, e := a.enforcer.GetImplicitRolesForUser(role)
		if e != nil {
			return false, ErrUnavailable
		}
		subjects = append(subjects, inherited...)
	}
	ok, e := a.enforcer.Enforce(subjects, operation)
	if e != nil {
		return false, ErrUnavailable
	}
	return ok, nil
}

func (a *Access) reload(initial bool) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	started := time.Now()
	e := a.enforcer.LoadPolicy()
	if e != nil {
		if a.available || initial {
			a.log.Error("management policy reload failed")
		}
		a.available = false
		return ErrUnavailable
	}
	p, e := a.enforcer.GetPolicy()
	if e != nil {
		a.available = false
		return ErrUnavailable
	}
	g, e := a.enforcer.GetGroupingPolicy()
	if e != nil {
		a.available = false
		return ErrUnavailable
	}
	cost := time.Since(started)
	if initial || !a.available {
		a.log.Info("management policy loaded", "p_count", len(p), "g_count", len(g), "load_duration", cost, "reload_interval", a.settings.Reload)
	}
	if initial && (len(p)+len(g) >= a.settings.WarnRules || cost >= time.Duration(float64(a.settings.Reload)*float64(a.settings.WarnLoadPercent)/100)) {
		a.log.Warn("consider a longer policy reload interval; this increases authorization change propagation delay", "rules", len(p)+len(g))
		if len(p)+len(g) >= a.settings.WarnRules*10 {
			a.log.Warn("review shared roles and scoped policy loading; longer intervals do not reduce resident policy size")
		}
	}
	a.available = true
	a.loadedAt = time.Now().UTC()
	return nil
}

// Run is explicitly owned/joined by assembly. Casbin's AutoLoad discards reload errors,
// so this periodic owner preserves the required observable fail-closed health boundary.
func (a *Access) Run(ctx context.Context) {
	ticker := time.NewTicker(a.settings.Reload)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = a.reload(false)
		}
	}
}

// Ready is detail-free; failed policy or credential reads are not fresh success.
func (a *Access) Ready() bool {
	a.mu.RLock()
	available := a.available
	a.mu.RUnlock()
	if !available {
		return false
	}
	_, e := loadPrincipals(a.settings.Credentials)
	return e == nil
}
