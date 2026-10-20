// SPDX-License-Identifier: Apache-2.0
package controllerclient

import (
	"context"
	"regexp"
	"sort"
	"strconv"
	"time"

	"github.com/PicketX/PicketX/pkg/configuration"
)

type Report struct {
	Scope                string    `json:"scope"`
	Authority            string    `json:"authority"`
	Revision             string    `json:"revision"`
	ObservedAt           time.Time `json:"observed_at"`
	Ready                *bool     `json:"ready,omitempty"`
	Nodes                []Node    `json:"nodes"`
	EnforcementConfirmed bool      `json:"enforcement_confirmed"`
}
type Node struct {
	AgentRef         string `json:"agent_ref"`
	DesiredRevision  string `json:"desired_revision"`
	ReportedRevision string `json:"reported_revision,omitempty"`
	State            string `json:"state"`
	FreshForMillis   int64  `json:"fresh_for_millis"`
}

func (r Report) Healthy() bool {
	if r.Ready == nil || !*r.Ready || len(r.Nodes) == 0 {
		return false
	}
	for _, n := range r.Nodes {
		if n.State != "reported-applied" {
			return false
		}
	}
	return true
}

type publication struct {
	Authority string `json:"authority"`
	Revision  uint64 `json:"revision"`
	Policy    string `json:"policy"`
	Facts     string `json:"facts"`
}

// Each independently bounded decoded input byte can occupy six JSON bytes
// (for example '<' becomes '\u003c'). Reserve the remaining bounded envelope.
const maxPublicationResponse = 2*configuration.MaxBytes*6 + 4096

func (p publication) validSizes() bool {
	return len(p.Policy) <= configuration.MaxBytes && len(p.Facts) <= configuration.MaxBytes
}

type observations struct {
	Authority  string        `json:"authority"`
	ObservedAt time.Time     `json:"observed_at"`
	Nodes      []observation `json:"nodes"`
}
type observation struct {
	AgentRef        string     `json:"agent_ref"`
	DesiredRevision string     `json:"desired_revision"`
	DesiredSHA256   string     `json:"desired_sha256"`
	Connection      string     `json:"connection"`
	Freshness       string     `json:"freshness"`
	FreshForMillis  int64      `json:"fresh_for_millis"`
	MatchesDesired  bool       `json:"matches_desired"`
	Report          *assertion `json:"report"`
}
type assertion struct {
	Revision   string    `json:"revision"`
	SHA256     string    `json:"sha256"`
	Phase      string    `json:"phase"`
	ObservedAt string    `json:"observed_at"`
	ReceivedAt time.Time `json:"received_at"`
}

var identity = regexp.MustCompile(`^[A-Za-z0-9_.:/-]{1,128}$`)
var hash = regexp.MustCompile(`^[a-f0-9]{64}$`)

func validID(s string) bool { return identity.MatchString(s) }
func validRevision(s string) bool {
	n, e := strconv.ParseUint(s, 10, 64)
	return e == nil && n > 0 && strconv.FormatUint(n, 10) == s
}

// Inspect samples authenticated Controller observations. It neither applies
// configuration nor independently checks a remote Agent's kernel or packets.
func Inspect(ctx context.Context, o Options, doctor bool) (Report, error) {
	c, e := newClient(o)
	if e != nil {
		return Report{}, e
	}
	defer c.http.CloseIdleConnections()
	ctx, cancel := context.WithTimeout(ctx, o.Timeout)
	defer cancel()
	var ready *bool
	if doctor {
		code, e := c.get(ctx, "/readyz", false, nil, 0)
		if e != nil {
			return Report{}, e
		}
		v := code == 200
		ready = &v
	}
	var before, after publication
	code, e := c.get(ctx, "/api/v1/publication", true, &before, maxPublicationResponse)
	if e != nil {
		return Report{}, e
	}
	if code != 200 {
		return Report{}, errRead
	}
	if !before.validSizes() {
		return Report{}, errData
	}
	started := time.Now()
	var view observations
	code, e = c.get(ctx, "/api/v1/agent-observations", true, &view, 4<<20)
	if e != nil {
		return Report{}, e
	}
	if code != 200 {
		return Report{}, errRead
	}
	code, e = c.get(ctx, "/api/v1/publication", true, &after, maxPublicationResponse)
	if e != nil {
		return Report{}, e
	}
	if code != 200 {
		return Report{}, errRead
	}
	if !after.validSizes() || !validID(before.Authority) || before.Revision == 0 || before.Authority != after.Authority || before.Revision != after.Revision || view.Authority != before.Authority || view.ObservedAt.IsZero() || view.Nodes == nil || len(view.Nodes) > 4096 {
		return Report{}, errData
	}
	out := Report{Scope: "controller-observations", Authority: before.Authority, Revision: strconv.FormatUint(before.Revision, 10), ObservedAt: view.ObservedAt, Ready: ready, Nodes: []Node{}, EnforcementConfirmed: false}
	seen := map[string]bool{}
	elapsed := time.Since(started).Milliseconds()
	for _, n := range view.Nodes {
		if !validID(n.AgentRef) || seen[n.AgentRef] || (n.Connection != "connected" && n.Connection != "disconnected") || (n.Freshness != "fresh" && n.Freshness != "stale" && n.Freshness != "unknown") || n.FreshForMillis < 0 || n.FreshForMillis > 60000 {
			return Report{}, errData
		}
		seen[n.AgentRef] = true
		if n.DesiredRevision != "" {
			if !validRevision(n.DesiredRevision) || n.DesiredRevision != out.Revision || !hash.MatchString(n.DesiredSHA256) {
				return Report{}, errData
			}
		} else if n.DesiredSHA256 != "" {
			return Report{}, errData
		}
		if r := n.Report; r != nil {
			if !validRevision(r.Revision) || !hash.MatchString(r.SHA256) || r.ReceivedAt.IsZero() || len(r.ObservedAt) > 128 || (r.Phase != "RECEIVED" && r.Phase != "APPLIED" && r.Phase != "FAILED") {
				return Report{}, errData
			}
			if n.MatchesDesired && (r.Revision != n.DesiredRevision || r.SHA256 != n.DesiredSHA256) {
				return Report{}, errData
			}
		} else if n.MatchesDesired || n.Freshness == "fresh" {
			return Report{}, errData
		}
		remaining := n.FreshForMillis - elapsed
		if remaining < 0 {
			remaining = 0
		}
		row := Node{AgentRef: n.AgentRef, DesiredRevision: n.DesiredRevision, State: "applying", FreshForMillis: remaining}
		if n.Report != nil {
			row.ReportedRevision = n.Report.Revision
		}
		switch {
		case n.DesiredRevision == "":
			row.State = "unbound"
		case n.Connection == "disconnected":
			row.State = "disconnected"
		case n.Report == nil:
			row.State = "unknown"
		case n.Freshness != "fresh" || remaining <= 0:
			row.State = "stale"
		case n.Report.Phase == "FAILED":
			row.State = "failed"
		case n.Report.Phase == "APPLIED" && n.MatchesDesired:
			row.State = "reported-applied"
		}
		if o.Agent == "" || o.Agent == row.AgentRef {
			out.Nodes = append(out.Nodes, row)
		}
	}
	if o.Agent != "" && len(out.Nodes) == 0 {
		return Report{}, errData
	}
	sort.Slice(out.Nodes, func(i, j int) bool { return out.Nodes[i].AgentRef < out.Nodes[j].AgentRef })
	return out, nil
}
