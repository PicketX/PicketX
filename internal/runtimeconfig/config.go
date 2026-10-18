// SPDX-License-Identifier: Apache-2.0
package runtimeconfig

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/PicketX/PicketX/internal/agentsync"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/v2"
	"go.yaml.in/yaml/v3"
)

type Config struct {
	HTTPListen            string
	Poll, Shutdown        time.Duration
	Database, Credentials string
	Source                struct{ Mode, Policy, Facts string }
	Management            struct {
		Credentials                                          string
		Reload                                               time.Duration
		WarnRules, WarnLoadPercent, RequestsPerSecond, Burst int
	}
	Agent struct {
		Enabled                            bool
		Listen, Authority, TLSCert, TLSKey string
		Limits                             agentsync.Limits
	}
}

func defaults() map[string]any {
	return map[string]any{
		"http.listen": "127.0.0.1:8080", "poll": "1s", "shutdown": "5s", "database": "", "credentials": "", "source.mode": "file", "source.policy": "", "source.facts": "",
		"management.credentials": "", "management.reload": "30s", "management.warn_rules": 10000, "management.warn_load_percent": 50, "management.requests_per_second": 128, "management.burst": 256,
		"agent.enabled": false, "agent.listen": "127.0.0.1:9090", "agent.authority": "", "agent.tls_cert": "", "agent.tls_key": "",
		"agent.max_credentials": 128, "agent.max_streams": 128, "agent.max_frame": 4 << 20, "agent.establish_per_second": 128, "agent.messages_per_second": 32, "agent.hello_timeout": "5s", "agent.send_timeout": "10s",
	}
}

// Load is the only runtime loading boundary. An explicitly present empty value
// overrides lower layers and is validated; absent flags never overwrite a file.
func Load(path string, env, overrides map[string]string) (Config, error) {
	var c Config
	k := koanf.New(".")
	defs := defaults()
	if e := k.Load(confmap.Provider(defs, "."), nil); e != nil {
		return c, e
	}
	if path != "" {
		info, e := os.Stat(path)
		if e != nil || !info.Mode().IsRegular() {
			return c, fmt.Errorf("runtime configuration must be a regular file")
		}
		f, e := os.Open(path)
		if e != nil {
			return c, fmt.Errorf("open runtime configuration: %w", e)
		}
		defer f.Close()
		info, e = f.Stat()
		if e != nil || !info.Mode().IsRegular() {
			return c, fmt.Errorf("runtime configuration must be a regular file")
		}
		data, e := io.ReadAll(io.LimitReader(f, (4<<20)+1))
		if e != nil || len(data) > 4<<20 {
			return c, fmt.Errorf("runtime configuration exceeds byte limit")
		}
		d := yaml.NewDecoder(bytes.NewReader(data))
		var n yaml.Node
		if e = d.Decode(&n); e != nil {
			return c, fmt.Errorf("invalid runtime YAML")
		}
		var extra yaml.Node
		if d.Decode(&extra) != io.EOF {
			return c, fmt.Errorf("expected one runtime document")
		}
		flat := map[string]any{}
		if len(n.Content) != 1 {
			return c, fmt.Errorf("invalid runtime document")
		}
		if e = flatten(n.Content[0], "", defs, flat); e != nil {
			return c, e
		}
		if e = k.Load(confmap.Provider(flat, "."), nil); e != nil {
			return c, e
		}
	}
	names := map[string]string{}
	for key := range defs {
		names["PICKETX_"+strings.ToUpper(strings.ReplaceAll(key, ".", "_"))] = key
	}
	for name, value := range env {
		if name == "PICKETX_CONFIG" || !strings.HasPrefix(name, "PICKETX_") {
			continue
		}
		key, ok := names[name]
		if !ok {
			return c, fmt.Errorf("unknown runtime environment key %s", name)
		}
		v, e := parseValue(value, defs[key])
		if e != nil {
			return c, fmt.Errorf("invalid runtime setting %s", key)
		}
		if e = k.Set(key, v); e != nil {
			return c, e
		}
	}
	for key, value := range overrides {
		sample, ok := defs[key]
		if !ok {
			return c, fmt.Errorf("unknown runtime flag key %s", key)
		}
		v, e := parseValue(value, sample)
		if e != nil {
			return c, fmt.Errorf("invalid runtime setting %s", key)
		}
		if e = k.Set(key, v); e != nil {
			return c, e
		}
	}
	c.HTTPListen = k.String("http.listen")
	c.Database = k.String("database")
	c.Credentials = k.String("credentials")
	c.Source.Policy = k.String("source.policy")
	c.Source.Mode = k.String("source.mode")
	c.Source.Facts = k.String("source.facts")
	c.Management.Credentials = k.String("management.credentials")
	c.Management.WarnRules = k.Int("management.warn_rules")
	c.Management.WarnLoadPercent = k.Int("management.warn_load_percent")
	c.Management.RequestsPerSecond = k.Int("management.requests_per_second")
	c.Management.Burst = k.Int("management.burst")
	c.Agent.Enabled = k.Bool("agent.enabled")
	c.Agent.Listen = k.String("agent.listen")
	c.Agent.Authority = k.String("agent.authority")
	c.Agent.TLSCert = k.String("agent.tls_cert")
	c.Agent.TLSKey = k.String("agent.tls_key")
	var e error
	for key, target := range map[string]*time.Duration{"poll": &c.Poll, "shutdown": &c.Shutdown, "management.reload": &c.Management.Reload, "agent.hello_timeout": &c.Agent.Limits.HelloTimeout, "agent.send_timeout": &c.Agent.Limits.SendTimeout} {
		*target, e = time.ParseDuration(k.String(key))
		if e != nil || *target <= 0 {
			return c, fmt.Errorf("invalid positive duration %s", key)
		}
	}
	c.Agent.Limits.MaxCredentials = k.Int("agent.max_credentials")
	c.Agent.Limits.MaxStreams = k.Int("agent.max_streams")
	c.Agent.Limits.MaxSnapshotBytes = k.Int("agent.max_frame")
	c.Agent.Limits.EstablishPerSecond = k.Int("agent.establish_per_second")
	c.Agent.Limits.MessagesPerSecond = k.Int("agent.messages_per_second")
	if e = c.Validate(); e != nil {
		return c, e
	}
	return c, nil
}
func (c Config) Validate() error {
	for _, addr := range []string{c.HTTPListen, c.Agent.Listen} {
		_, p, e := net.SplitHostPort(addr)
		if e != nil {
			return fmt.Errorf("invalid listen address")
		}
		port, e := strconv.Atoi(p)
		if e != nil || port < 0 || port > 65535 {
			return fmt.Errorf("invalid listen port")
		}
	}
	if c.Poll <= 0 || c.Shutdown <= 0 {
		return fmt.Errorf("invalid lifecycle duration")
	}
	if (c.Agent.TLSCert == "") != (c.Agent.TLSKey == "") {
		return fmt.Errorf("Agent TLS certificate and key must be paired")
	}
	authority := c.Agent.Authority
	if authority == "" && !c.Agent.Enabled {
		authority = "disabled"
	}
	s, e := agentsync.New(authority, c.Agent.Limits)
	if e != nil {
		return e
	}
	s.Close()
	if c.Agent.Enabled && (c.Database == "" || c.Credentials == "") {
		return fmt.Errorf("enabled Agent service requires database and credentials")
	}
	if c.Source.Mode != "file" && c.Source.Mode != "database" {
		return fmt.Errorf("invalid source mode")
	}
	if c.Source.Mode == "file" {
		if c.Management.Credentials != "" {
			return fmt.Errorf("management credentials require database source mode")
		}
		if c.Agent.Enabled && (c.Source.Policy == "" || c.Source.Facts == "") {
			return fmt.Errorf("file mode requires policy and facts")
		}
	} else if !c.Agent.Enabled || c.Source.Policy != "" || c.Source.Facts != "" || c.Management.Credentials == "" {
		return fmt.Errorf("database source mode requires Agent service and management credentials, without policy/facts files")
	}
	if c.Management.Reload <= 0 || c.Management.Reload > 24*time.Hour || c.Management.WarnRules < 1 || c.Management.WarnRules > 100000000 || c.Management.WarnLoadPercent < 1 || c.Management.WarnLoadPercent > 1000 || c.Management.RequestsPerSecond < 1 || c.Management.RequestsPerSecond > 100000 || c.Management.Burst < 1 || c.Management.Burst > 100000 {
		return fmt.Errorf("invalid management lifecycle or limits")
	}
	return nil
}
func parseValue(s string, sample any) (any, error) {
	switch sample.(type) {
	case bool:
		return strconv.ParseBool(s)
	case int:
		return strconv.Atoi(s)
	default:
		return s, nil
	}
}
func flatten(n *yaml.Node, prefix string, defs, out map[string]any) error {
	if n.Kind != yaml.MappingNode {
		return fmt.Errorf("runtime root/group must be a mapping")
	}
	seen := map[string]bool{}
	for i := 0; i < len(n.Content); i += 2 {
		key, value := n.Content[i], n.Content[i+1]
		if key.Kind != yaml.ScalarNode || key.Tag != "!!str" || seen[key.Value] {
			return fmt.Errorf("duplicate or invalid runtime key")
		}
		seen[key.Value] = true
		path := key.Value
		if prefix != "" {
			path = prefix + "." + path
		}
		sample, leaf := defs[path]
		if value.Kind == yaml.MappingNode {
			known := false
			for k := range defs {
				if strings.HasPrefix(k, path+".") {
					known = true
					break
				}
			}
			if !known || leaf {
				return fmt.Errorf("unknown runtime group %s", path)
			}
			if e := flatten(value, path, defs, out); e != nil {
				return e
			}
			continue
		}
		if !leaf || value.Kind != yaml.ScalarNode || value.Tag == "!!null" {
			return fmt.Errorf("unknown or invalid runtime setting %s", path)
		}
		expected := "!!str"
		switch sample.(type) {
		case bool:
			expected = "!!bool"
		case int:
			expected = "!!int"
		}
		if value.Tag != expected {
			return fmt.Errorf("invalid runtime type %s", path)
		}
		v, e := parseValue(value.Value, sample)
		if e != nil {
			return fmt.Errorf("invalid runtime value %s", path)
		}
		if _, exists := out[path]; exists {
			return fmt.Errorf("duplicate runtime setting %s", path)
		}
		out[path] = v
	}
	return nil
}
