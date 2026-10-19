// SPDX-License-Identifier: Apache-2.0
package httpauth

import (
	"fmt"
	"slices"
	"strings"

	"github.com/PicketX/PicketX/pkg/core"
)

type Route struct {
	ID                       string
	Resource, Endpoint, Host string
	Methods, Paths           []string
	PathPrefixes             []string
}
type Engine struct {
	network *core.Engine
	routes  map[string]Route
}

func Compile(snapshot core.Snapshot, routes []Route) (*Engine, error) {
	n, e := core.Compile(snapshot)
	if e != nil {
		return nil, e
	}
	out := &Engine{network: n, routes: map[string]Route{}}
	for _, r := range routes {
		if !ValidID(r.ID) || !validHost(r.Host) || len(r.Methods) == 0 || len(r.Methods) > 7 || len(r.Paths)+len(r.PathPrefixes) == 0 || len(r.Paths)+len(r.PathPrefixes) > 128 || (r.Paths != nil && len(r.Paths) == 0) || (r.PathPrefixes != nil && len(r.PathPrefixes) == 0) {
			return nil, fmt.Errorf("invalid HTTP route")
		}
		if _, ok := out.routes[r.ID]; ok {
			return nil, fmt.Errorf("duplicate HTTP route")
		}
		found := false
		for _, res := range snapshot.Resources {
			if res.Name == r.Resource {
				for _, ep := range res.Endpoints {
					if ep.Name == r.Endpoint {
						found = true
					}
				}
			}
		}
		if !found {
			return nil, fmt.Errorf("unknown HTTP route target")
		}
		methods, paths := map[string]bool{}, map[string]bool{}
		for _, m := range r.Methods {
			if !validMethod(m) || methods[m] {
				return nil, fmt.Errorf("invalid/duplicate route method")
			}
			methods[m] = true
		}
		for _, p := range r.Paths {
			v, e := requestPath(p)
			if e != nil || p != v || paths[p] {
				return nil, fmt.Errorf("invalid/duplicate route path")
			}
			paths[p] = true
		}
		prefixes := map[string]bool{}
		for _, p := range r.PathPrefixes {
			v, err := requestPath(p)
			if err != nil || p != v || (p != "/" && strings.HasSuffix(p, "/")) || prefixes[p] {
				return nil, fmt.Errorf("invalid/duplicate route path prefix")
			}
			prefixes[p] = true
		}
		r.Methods = slices.Clone(r.Methods)
		r.Paths = slices.Clone(r.Paths)
		r.PathPrefixes = slices.Clone(r.PathPrefixes)
		out.routes[r.ID] = r
	}
	return out, nil
}
func (e *Engine) HasRoute(id string) bool {
	if e == nil {
		return false
	}
	_, ok := e.routes[id]
	return ok
}
func (e *Engine) Decide(q Query) (core.Decision, error) {
	if err := Validate(q); err != nil {
		return core.Decision{}, err
	}
	if e == nil || e.network == nil {
		return core.Decision{}, fmt.Errorf("HTTP engine not compiled")
	}
	r, ok := e.routes[q.Route]
	if !ok {
		return core.Decision{Reason: "unknown-route"}, nil
	}
	p, _ := requestPath(q.URI)
	match := slices.Contains(r.Paths, p)
	for _, prefix := range r.PathPrefixes {
		if prefix == "/" || p == prefix || strings.HasPrefix(p, prefix+"/") {
			match = true
			break
		}
	}
	if q.Host != r.Host || !slices.Contains(r.Methods, q.Method) || !match {
		return core.Decision{Reason: "route-mismatch"}, nil
	}
	return e.network.Decide(core.Query{Source: q.Source, ResourceName: r.Resource, EndpointName: r.Endpoint, Action: "connect", At: q.At})
}

// SameTargets prevents a retained route identity being silently rebound.
func (e *Engine) SameTargets(next *Engine) error {
	for id, old := range e.routes {
		if r, ok := next.routes[id]; ok && (r.Resource != old.Resource || r.Endpoint != old.Endpoint || r.Host != old.Host) {
			return fmt.Errorf("HTTP route UID cannot change target")
		}
	}
	return nil
}
