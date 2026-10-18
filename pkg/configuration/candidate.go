// SPDX-License-Identifier: Apache-2.0
package configuration

import (
	"encoding/json"
	"fmt"
	"slices"

	"github.com/PicketX/PicketX/pkg/httpauth"
	"github.com/PicketX/PicketX/pkg/networkplan"
)

// ObjectIdentity maps human names to stable IDs used in planner results.
type ObjectIdentity struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
	UID  string `json:"uid"`
}

// Candidate is immutable accepted desired intent. It owns no kernel state.
type Candidate struct {
	http    *httpauth.Engine
	plan    *networkplan.Plan
	input   []byte
	hash    string
	objects []ObjectIdentity
}

func Compile(b *Bundle, f Facts) (*Candidate, error) {
	in, e := candidateInput(b, f)
	if e != nil {
		return nil, e
	}
	p, e := networkplan.Compile(in)
	if e != nil {
		return nil, e
	}
	return newCandidate(b, in, p)
}
func Replace(previous *Candidate, b *Bundle, f Facts) (*Candidate, []networkplan.Withdrawal, error) {
	if previous == nil || previous.plan == nil {
		return nil, nil, fmt.Errorf("previous candidate is not compiled")
	}
	in, e := candidateInput(b, f)
	if e != nil {
		return nil, nil, e
	}
	kinds := map[string]string{}
	for _, o := range previous.objects {
		kinds[o.UID] = o.Kind
	}
	for _, o := range b.objects {
		if kind, ok := kinds[o.Metadata.UID]; ok && kind != o.Kind {
			return nil, nil, fmt.Errorf("UID %q cannot change kind", o.Metadata.UID)
		}
	}
	r, e := networkplan.Replace(previous.plan, in)
	if e != nil {
		return nil, nil, e
	}
	c, e := newCandidate(b, in, r.Plan)
	if e != nil {
		return nil, nil, e
	}
	if e = previous.http.SameTargets(c.http); e != nil {
		return nil, nil, e
	}
	return c, r.Withdrawals, nil
}
func candidateInput(b *Bundle, f Facts) (networkplan.Input, error) {
	if b == nil || len(b.objects) == 0 || b.hash == "" {
		return networkplan.Input{}, fmt.Errorf("configuration bundle is not decoded")
	}
	in, e := translate(b, f)
	if e != nil {
		return networkplan.Input{}, e
	}
	// Detach every map/slice from policy and caller-owned runtime facts.
	data, e := json.Marshal(in)
	if e != nil {
		return networkplan.Input{}, e
	}
	var owned networkplan.Input
	e = json.Unmarshal(data, &owned)
	return owned, e
}
func newCandidate(b *Bundle, in networkplan.Input, p *networkplan.Plan) (*Candidate, error) {
	data, e := json.Marshal(in)
	if e != nil {
		return nil, e
	}
	c := &Candidate{plan: p, input: data, hash: b.hash}
	var routes []httpauth.Route
	for _, o := range b.objects {
		s, ok := o.Spec.(httpRouteSpec)
		if !ok {
			continue
		}
		found := false
		for _, obj := range b.objects {
			rs, ok := obj.Spec.(resourceSpec)
			if !ok || obj.Metadata.Name != s.ResourceRef.Name || obj.Metadata.UID != s.ResourceRef.UID {
				continue
			}
			for _, ep := range rs.Endpoints {
				if ep.Name == s.EndpointName && ep.Network.Protocol == "TCP" {
					found = true
				}
			}
		}
		if !found {
			return nil, fmt.Errorf("HTTPRoute requires pinned TCP resource endpoint")
		}
		routes = append(routes, httpauth.Route{ID: o.Metadata.UID, Resource: s.ResourceRef.UID, Endpoint: s.EndpointName, Host: s.Host, Methods: s.Methods, Paths: s.Paths})
	}
	c.http, e = httpauth.Compile(in.Logical, routes)
	if e != nil {
		return nil, e
	}
	for _, o := range b.objects {
		c.objects = append(c.objects, ObjectIdentity{o.Kind, o.Metadata.Name, o.Metadata.UID})
	}
	return c, nil
}
func (c *Candidate) HTTP() *httpauth.Engine    { return c.http }
func (c *Candidate) Plan() *networkplan.Plan   { return c.plan }
func (c *Candidate) Hash() string              { return c.hash }
func (c *Candidate) Objects() []ObjectIdentity { return slices.Clone(c.objects) }

// Input returns an independently mutable complete input, retaining future and
// expired contributions rather than replacing them with a current-time view.
func (c *Candidate) Input() networkplan.Input {
	var in networkplan.Input
	if c == nil {
		return in
	}
	// These bytes were encoded from valid typed input and are never exposed.
	if e := json.Unmarshal(c.input, &in); e != nil {
		panic("configuration: corrupted private candidate input")
	}
	return in
}
