// SPDX-License-Identifier: Apache-2.0
package controllerapp

import (
	"crypto/sha256"
	"database/sql"
	"errors"
	"io"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/PicketX/PicketX/internal/accessrequest"
	"github.com/PicketX/PicketX/internal/agentsync"
	"github.com/PicketX/PicketX/internal/publication"
	"github.com/PicketX/PicketX/pkg/configuration"
	"gorm.io/gorm"
)

// A completed transaction must not need another fallible request-row query
// before publishing its committed candidate. Exercise real SQLite and service.
func TestAccessRequestNoFalliblePostCommitReadback(t *testing.T) {
	for _, operation := range []string{"create", "approve", "reject"} {
		t.Run(operation, func(t *testing.T) {
			c := managedConfig(t)
			s, err := publication.OpenMode(c.Database, c.Agent.Authority, "database")
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()
			svc, err := agentsync.New(c.Agent.Authority, c.Agent.Limits)
			if err != nil {
				t.Fatal(err)
			}
			defer svc.Close()
			if err = svc.ReplaceCredentials([]agentsync.Credential{{ID: "test", AgentRef: "agent-a", SHA256: sha256.Sum256([]byte(token))}}); err != nil {
				t.Fatal(err)
			}
			policy, _ := os.ReadFile("../../examples/static/gitea.yaml")
			facts, _ := os.ReadFile("../../examples/static/facts.json")
			if _, err = s.AcceptManaged(0, "seed", "seed", policy, facts, svc.CheckPublication); err != nil {
				t.Fatal(err)
			}
			candidate, revision := s.Current()
			if err = svc.Publish(candidate, revision); err != nil {
				t.Fatal(err)
			}
			p := &publisher{cfg: c, store: s, service: svc, log: slog.New(slog.NewTextHandler(io.Discard, nil))}
			p.ready.Store(true)
			r := accessrequest.Record{UID: "request-readback", Requester: "principal:requester", State: "pending", Version: 1, ExpiresAt: time.Now().Add(time.Hour).UTC(), CreatedAt: time.Now().UTC()}
			if operation != "create" {
				r, err = p.CreateRequest(1, r)
				if err != nil {
					t.Fatal(err)
				}
			}
			armed := false
			err = s.GORM().Callback().Raw().After("gorm:raw").Register("test:arm_committed_request", func(tx *gorm.DB) {
				q := tx.Statement.SQL.String()
				if strings.HasPrefix(q, "INSERT INTO controller_request_event") || strings.HasPrefix(q, "UPDATE controller_publication SET revision") {
					armed = true
				}
			})
			if err != nil {
				t.Fatal(err)
			}
			err = s.GORM().Callback().Row().Before("gorm:row").Register("test:fail_postcommit_request_read", func(tx *gorm.DB) {
				_, inTransaction := tx.Statement.ConnPool.(*sql.Tx)
				if armed && !inTransaction && strings.HasPrefix(tx.Statement.SQL.String(), "SELECT id,data FROM controller_access_request") {
					tx.AddError(errors.New("post-commit readback unavailable"))
				}
			})
			if err != nil {
				t.Fatal(err)
			}
			if operation == "create" {
				r, err = p.CreateRequest(1, r)
			} else {
				r.Version++
				r.Approver = "principal:approver"
				now := time.Now().UTC()
				r.DecidedAt = &now
				r.DecisionReason = "independent"
				r.State = "rejected"
				approvedPolicy := ""
				if operation == "approve" {
					r.State = "approved"
					r.GrantUID = "grant-readback"
					r.ApprovedExpiresAt = &r.ExpiresAt
					r.Revision = 2
					b, _ := configuration.Decode(policy)
					added, e := b.AddGrant(configuration.GrantInput{Name: r.UID, UID: r.GrantUID, Source: "2001:db8::42", ResourceName: "gitea", ResourceUID: "resource-gitea-v1", EndpointNames: []string{"https"}, ExpiresAt: r.ExpiresAt})
					if e != nil {
						t.Fatal(e)
					}
					approvedPolicy = string(added)
				}
				r, err = p.DecideAccessRequest(1, r, approvedPolicy, string(facts))
			}
			if err != nil || !armed || r.ID == 0 || p.failed || !p.ready.Load() {
				t.Fatal("committed operation disrupted by unnecessary readback", r.ID, err, armed, p.failed)
			}
			wantRevision := uint64(1)
			if operation == "approve" {
				wantRevision = 2
			}
			observed, err := svc.Observations()
			if err != nil || len(observed.Nodes) != 1 || observed.Nodes[0].DesiredRevision != strconv.FormatUint(wantRevision, 10) {
				t.Fatal("committed publication not delivered", observed, err)
			}
			if err = s.Close(); err != nil {
				t.Fatal(err)
			}
			s, err = publication.OpenMode(c.Database, c.Agent.Authority, "database")
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()
			got, err := s.GetRequest(r.UID)
			if err != nil || got.State != r.State || got.ID != r.ID || got.GrantUID != r.GrantUID {
				t.Fatal("durable request differs", got, r, err)
			}
		})
	}
}
