// SPDX-License-Identifier: Apache-2.0
package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"slices"
	"time"

	"github.com/PicketX/PicketX/pkg/configuration"
	"github.com/PicketX/PicketX/pkg/networkplan"
	"github.com/spf13/cobra"
)

// New assembles local operator commands. No command opens listeners or applies rules.
func New(out, errOut io.Writer) *cobra.Command {
	root := &cobra.Command{Use: "picketx", Short: "PicketX configuration diagnostics", SilenceErrors: true, SilenceUsage: true}
	root.SetOut(out)
	root.SetErr(errOut)
	config := &cobra.Command{Use: "config", Short: "Validate desired network configuration"}
	var path, factsPath, atText string
	check := &cobra.Command{Use: "check", Short: "Check a complete policy against operator-supplied facts (no enforcement)", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		at := time.Now().UTC()
		if atText != "" {
			var e error
			at, e = time.Parse(time.RFC3339Nano, atText)
			if e != nil || at.IsZero() {
				return fmt.Errorf("--at requires a nonzero RFC3339 timestamp")
			}
		}
		policy, e := readBounded(path)
		if e != nil {
			return fmt.Errorf("read policy: %w", e)
		}
		b, e := configuration.Decode(policy)
		if e != nil {
			return fmt.Errorf("decode policy: %w", e)
		}
		data, e := readBounded(factsPath)
		if e != nil {
			return fmt.Errorf("read facts: %w", e)
		}
		f, e := decodeFacts(data)
		if e != nil {
			return fmt.Errorf("decode diagnostic facts: %w", e)
		}
		candidate, e := configuration.Compile(b, f)
		if e != nil {
			return fmt.Errorf("compile candidate: %w", e)
		}
		projections := make([]networkplan.Projection, 0)
		// AgentRefs lists protected targets only. Diagnostics must also show an
		// explicit empty projection for known unbound Agents after withdrawal.
		var agents []string
		for _, agent := range candidate.Input().Agents {
			agents = append(agents, agent.Name)
		}
		slices.Sort(agents)
		for _, a := range agents {
			p, e := candidate.Plan().Projection(a, at)
			if e != nil {
				return e
			}
			projections = append(projections, p)
		}
		diagnostics, e := candidate.Plan().Diagnostics()
		if e != nil {
			return e
		}
		report := struct {
			State                string                         `json:"state"`
			EnforcementConfirmed bool                           `json:"enforcementConfirmed"`
			PolicyHash           string                         `json:"policyHash"`
			At                   time.Time                      `json:"at"`
			Objects              []configuration.ObjectIdentity `json:"objects"`
			Projections          []networkplan.Projection       `json:"projections"`
			Diagnostics          []networkplan.Diagnostic       `json:"diagnostics"`
		}{"validated", false, candidate.Hash(), at.UTC(), candidate.Objects(), projections, diagnostics}
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(report)
	}}
	check.Flags().StringVar(&path, "file", "", "Complete multi-document YAML publication")
	check.Flags().StringVar(&factsPath, "facts", "", "Operator-supplied diagnostic facts JSON (not live evidence)")
	check.Flags().StringVar(&atText, "at", "", "Evaluate at an absolute RFC3339 instant (default: now)")
	_ = check.MarkFlagRequired("file")
	_ = check.MarkFlagRequired("facts")
	config.AddCommand(check)
	root.AddCommand(config)
	root.AddCommand(passwordCommand())
	root.AddCommand(controllerInitCommand())
	root.AddCommand(nginxInitCommand())
	return root
}
