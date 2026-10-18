// SPDX-License-Identifier: Apache-2.0
package configuration

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/PicketX/PicketX/pkg/core"
)

var ErrGrantNotFound = errors.New("Grant not found")

// Catalog views are detached desired intent, not effective or applied permission.
type ResourceView struct {
	Name      string         `json:"name"`
	UID       string         `json:"uid"`
	Endpoints []EndpointView `json:"endpoints"`
}
type EndpointView struct {
	Name     string `json:"name"`
	Protocol string `json:"protocol"`
	Port     uint16 `json:"port"`
}
type GrantScope struct {
	ResourceName  string   `json:"resource_name"`
	ResourceUID   string   `json:"resource_uid"`
	Selection     string   `json:"selection"`
	EndpointNames []string `json:"endpoint_names,omitempty"`
	Selector      string   `json:"selector,omitempty"`
}
type GrantView struct {
	Name      string       `json:"name"`
	UID       string       `json:"uid"`
	Source    string       `json:"source"`
	Scopes    []GrantScope `json:"scopes"`
	Mode      string       `json:"mode"`
	NotBefore string       `json:"not_before,omitempty"`
	ExpiresAt string       `json:"expires_at,omitempty"`
	Revoked   bool         `json:"revoked"`
}

func (b *Bundle) GrantCatalog() ([]ResourceView, []GrantView) {
	resources, grants := []ResourceView{}, []GrantView{}
	for _, o := range b.objects {
		switch s := o.Spec.(type) {
		case resourceSpec:
			r := ResourceView{Name: o.Metadata.Name, UID: o.Metadata.UID, Endpoints: []EndpointView{}}
			for _, e := range s.Endpoints {
				r.Endpoints = append(r.Endpoints, EndpointView{e.Name, e.Network.Protocol, e.Network.Port})
			}
			resources = append(resources, r)
		case contributionSpec:
			if o.Kind != "Grant" {
				continue
			}
			source, _ := core.ParseSource(s.Subject.CIDR)
			g := GrantView{Name: o.Metadata.Name, UID: o.Metadata.UID, Source: source.String(), Mode: s.Validity.Mode, Revoked: s.Revoked, Scopes: []GrantScope{}}
			if s.Validity.NotBefore != nil {
				g.NotBefore = *s.Validity.NotBefore
			}
			if s.Validity.ExpiresAt != nil {
				g.ExpiresAt = *s.Validity.ExpiresAt
			}
			for _, sc := range s.Scopes {
				v := GrantScope{ResourceName: sc.ResourceRef.Name, ResourceUID: sc.ResourceRef.UID, Selection: "all", EndpointNames: slices.Clone(sc.EndpointNames)}
				if sc.EndpointNames != nil {
					v.Selection = "named"
				}
				if sc.EndpointSelector != nil {
					v.Selection = "selector"
					selectorJSON, _ := json.Marshal(sc.EndpointSelector)
					v.Selector = string(selectorJSON)
				}
				g.Scopes = append(g.Scopes, v)
			}
			grants = append(grants, g)
		}
	}
	slices.SortFunc(grants, func(a, b GrantView) int { return strings.Compare(a.UID, b.UID) })
	return resources, grants
}

type GrantInput struct {
	Name, UID, Source, ResourceName, ResourceUID string
	EndpointNames                                []string
	ExpiresAt                                    time.Time
}

// AddGrant never mutates this Bundle. The caller must compile/Replace the complete
// returned publication with its real facts and previous candidate before accepting.
func (b *Bundle) AddGrant(in GrantInput) ([]byte, error) {
	if strings.TrimSpace(in.Name) == "" || len(in.Name) > 128 || !uidPattern.MatchString(in.UID) || in.ExpiresAt.IsZero() {
		return nil, fmt.Errorf("invalid Grant identity or expiry")
	}
	source, err := core.ParseSource(in.Source)
	if err != nil {
		return nil, err
	}
	if len(in.EndpointNames) == 0 {
		return nil, fmt.Errorf("explicit endpoint names required")
	}
	endpoints := map[string]bool{}
	found := false
	for _, o := range b.objects {
		if o.Metadata.UID == in.UID || (o.Kind == "Grant" && o.Metadata.Name == in.Name) {
			return nil, fmt.Errorf("duplicate Grant identity")
		}
		if o.Kind == "Resource" && o.Metadata.Name == in.ResourceName && o.Metadata.UID == in.ResourceUID {
			found = true
			for _, e := range o.Spec.(resourceSpec).Endpoints {
				endpoints[e.Name] = true
			}
		}
	}
	if !found {
		return nil, fmt.Errorf("pinned Resource missing")
	}
	for _, name := range in.EndpointNames {
		if !endpoints[name] {
			return nil, fmt.Errorf("missing or repeated endpoint")
		}
		delete(endpoints, name)
	}
	expires := in.ExpiresAt.UTC().Format(time.RFC3339Nano)
	newGrant := document{APIVersion, "Grant", metadata{Name: in.Name, UID: in.UID}, contributionSpec{Subject: subject{"source", source.String()}, Scopes: []scope{{ResourceRef: resourceRef{in.ResourceName, in.ResourceUID}, EndpointNames: slices.Clone(in.EndpointNames), Actions: []string{"connect"}}}, Validity: validity{Mode: "bounded", ExpiresAt: &expires}}}
	var out bytes.Buffer
	out.Write(b.source)
	out.WriteString("\n---\n")
	encoder := json.NewEncoder(&out)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(newGrant); err != nil {
		return nil, err
	}
	if _, err := Decode(out.Bytes()); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func (b *Bundle) RevokeGrant(uid string) ([]byte, error) {
	objects := slices.Clone(b.objects)
	for i, o := range objects {
		if o.Kind != "Grant" || o.Metadata.UID != uid {
			continue
		}
		s := o.Spec.(contributionSpec)
		if s.Revoked {
			return slices.Clone(b.source), nil
		}
		s.Revoked = true
		objects[i].Spec = s
		out, err := revokeSource(b.source, uid)
		if err != nil {
			return nil, err
		}
		next, err := Decode(out)
		if err != nil {
			return nil, err
		}
		// Parser coordinates are only an editing aid. Strictly decode again and
		// require the complete semantic result to equal exactly one revoked flag.
		expected, err := json.Marshal(objects)
		if err != nil {
			return nil, err
		}
		actual, err := json.Marshal(next.objects)
		if err != nil || !bytes.Equal(expected, actual) {
			return nil, fmt.Errorf("revocation changed unrelated policy")
		}
		return out, nil
	}
	return nil, ErrGrantNotFound
}
