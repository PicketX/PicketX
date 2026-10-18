// SPDX-License-Identifier: Apache-2.0
// Package configuration decodes the evolving v1alpha1 network policy contract.
// A validated or compiled candidate is desired intent, never proof of enforcement.
package configuration

import (
	"encoding/json"
	"github.com/PicketX/PicketX/pkg/core"
)

const APIVersion = "picketx.io/v1alpha1"
const MaxBytes = 4 << 20
const MaxObjects = 4096
const maxDepth = 64

type metadata struct {
	Name        string            `json:"name"`
	UID         string            `json:"uid"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}
type header struct {
	APIVersion string          `json:"apiVersion"`
	Kind       string          `json:"kind"`
	Metadata   metadata        `json:"metadata"`
	Spec       json.RawMessage `json:"spec"`
}
type document struct {
	APIVersion string   `json:"apiVersion"`
	Kind       string   `json:"kind"`
	Metadata   metadata `json:"metadata"`
	Spec       any      `json:"spec"`
}
type resourceSpec struct {
	Endpoints []endpoint `json:"endpoints"`
}
type endpoint struct {
	Name    string            `json:"name"`
	Type    *string           `json:"type,omitempty"`
	Labels  map[string]string `json:"labels,omitempty"`
	Network network           `json:"network"`
}
type network struct {
	Protocol  string   `json:"protocol"`
	Port      uint16   `json:"port"`
	Addresses []string `json:"addresses,omitempty"`
}
type resourceRef struct {
	Name string `json:"name"`
	UID  string `json:"uid"`
}
type selection struct {
	ResourceRef      resourceRef `json:"resourceRef"`
	EndpointNames    []string    `json:"endpointNames,omitempty"`
	EndpointSelector *selector   `json:"endpointSelector,omitempty"`
}
type scope struct {
	ResourceRef      resourceRef `json:"resourceRef"`
	EndpointNames    []string    `json:"endpointNames,omitempty"`
	EndpointSelector *selector   `json:"endpointSelector,omitempty"`
	Actions          []string    `json:"actions"`
}
type selector struct {
	MatchLabels      map[string]string `json:"matchLabels,omitempty"`
	MatchExpressions []requirement     `json:"matchExpressions,omitempty"`
}
type requirement struct {
	Key      string                `json:"key"`
	Operator core.SelectorOperator `json:"operator"`
	Values   []string              `json:"values,omitempty"`
}
type contributionSpec struct {
	Subject  subject  `json:"subject"`
	Scopes   []scope  `json:"scopes"`
	Validity validity `json:"validity"`
	Revoked  bool     `json:"revoked,omitempty"`
}
type subject struct {
	Type string `json:"type"`
	CIDR string `json:"cidr"`
}
type validity struct {
	Mode      string  `json:"mode"`
	NotBefore *string `json:"notBefore,omitempty"`
	ExpiresAt *string `json:"expiresAt,omitempty"`
}
type agentRef struct {
	Name string `json:"name"`
}
type agentSelection struct {
	AgentRef      *agentRef `json:"agentRef,omitempty"`
	AgentSelector *selector `json:"agentSelector,omitempty"`
}
type bindingSpec struct {
	Resources   []selection      `json:"resources"`
	Agents      []agentSelection `json:"agents"`
	Enforcement enforcement      `json:"enforcement"`
}
type enforcement struct {
	ProfileRef          string   `json:"profileRef"`
	Class               string   `json:"class"`
	Stage               string   `json:"stage"`
	AllowedDestinations []string `json:"allowedDestinations"`
	Handling            string   `json:"handling"`
}
type emptySpec struct {
	Empty bool `json:"empty"`
}
type httpRouteSpec struct {
	ResourceRef  resourceRef `json:"resourceRef"`
	EndpointName string      `json:"endpointName"`
	Host         string      `json:"host"`
	Methods      []string    `json:"methods"`
	Paths        []string    `json:"paths"`
}

// Bundle owns decoded policy and cannot be changed through exported fields.
type Bundle struct {
	objects []document
	hash    string
}
