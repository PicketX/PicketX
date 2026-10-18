// SPDX-License-Identifier: Apache-2.0
package controllerapp

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/PicketX/PicketX/internal/agentsync"
	"github.com/PicketX/PicketX/internal/publication"
	"github.com/PicketX/PicketX/internal/runtimeconfig"
	"github.com/PicketX/PicketX/internal/strictjson"
	"io"
	"log/slog"
	"os"
	"sync/atomic"
)

func readFile(path string, private bool) ([]byte, error) {
	info, e := os.Stat(path)
	if e != nil || !info.Mode().IsRegular() || (private && info.Mode().Perm()&0077 != 0) {
		return nil, fmt.Errorf("input must be a regular file with required permissions")
	}
	f, e := os.Open(path)
	if e != nil {
		return nil, e
	}
	defer f.Close()
	info, e = f.Stat()
	if e != nil || !info.Mode().IsRegular() || (private && info.Mode().Perm()&0077 != 0) {
		return nil, fmt.Errorf("input must be a regular file with required permissions")
	}
	data, e := io.ReadAll(io.LimitReader(f, (4<<20)+1))
	if e != nil {
		return nil, e
	}
	if len(data) > 4<<20 {
		return nil, fmt.Errorf("input exceeds byte limit")
	}
	return data, nil
}
func decodeCredentials(data []byte) ([]agentsync.Credential, error) {
	type entry struct {
		ID       string `json:"id"`
		AgentRef string `json:"agent_ref"`
		SHA256   string `json:"sha256"`
	}
	var dto struct {
		Credentials *[]entry `json:"credentials"`
	}
	if e := strictjson.Decode(data, &dto); e != nil || dto.Credentials == nil {
		return nil, fmt.Errorf("invalid credential document")
	}
	var out []agentsync.Credential
	for _, r := range *dto.Credentials {
		raw, e := hex.DecodeString(r.SHA256)
		if e != nil || len(raw) != 32 {
			return nil, fmt.Errorf("invalid credential digest")
		}
		var key [32]byte
		copy(key[:], raw)
		out = append(out, agentsync.Credential{ID: r.ID, AgentRef: r.AgentRef, SHA256: key})
	}
	return out, nil
}

type publisher struct {
	cfg            runtimeconfig.Config
	service        AgentService
	store          *publication.Store
	log            *slog.Logger
	ready          atomic.Bool
	credentialHash [32]byte
	problem        string
}

func (p *publisher) credentials() bool {
	raw, e := readFile(p.cfg.Credentials, true)
	if e != nil {
		p.problem = "credentials_read"
		return false
	}
	sum := sha256.Sum256(raw)
	if sum == p.credentialHash {
		return true
	}
	creds, e := decodeCredentials(raw)
	if e != nil {
		p.problem = "credentials_invalid"
		return false
	}
	if e = p.service.ReplaceCredentials(creds); e != nil {
		p.problem = "credentials_rejected"
		return false
	}
	p.credentialHash = sum
	return true
}
func (p *publisher) reload() error {
	previousProblem := p.problem
	p.problem = ""
	credentialsOK := p.credentials()
	policy, e := readFile(p.cfg.Source.Policy, false)
	if e != nil {
		p.problem = "policy_read"
	} else {
		facts, e := readFile(p.cfg.Source.Facts, false)
		if e != nil {
			p.problem = "facts_read"
		} else {
			changed, e := p.store.Accept(policy, facts, p.service.CheckPublication)
			if e != nil {
				var persistence *publication.PersistenceError
				if errors.As(e, &persistence) {
					return e
				}
				p.problem = "publication_rejected"
			} else if changed {
				candidate, revision := p.store.Current()
				if e = p.service.Publish(candidate, revision); e != nil {
					return fmt.Errorf("distribute durable publication: %w", e)
				}
				p.log.Info("accepted desired publication", "revision", revision)
			}
		}
	}
	p.ready.Store(credentialsOK && p.problem == "")
	if p.problem != "" && p.problem != previousProblem {
		p.log.Warn("runtime input rejected", "code", p.problem)
	} else if p.problem == "" && previousProblem != "" {
		p.log.Info("runtime inputs recovered")
	}
	return nil
}
