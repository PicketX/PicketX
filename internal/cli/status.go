// SPDX-License-Identifier: Apache-2.0
package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/PicketX/PicketX/internal/controllerclient"
	"github.com/spf13/cobra"
)

func statusCommand(doctor bool) *cobra.Command {
	var options controllerclient.Options
	var format string
	name, short := "status", "Read desired state and current Agent assertions (not independent enforcement proof)"
	if doctor {
		name, short = "doctor", "Check Controller readiness and fresh matching Agent assertions (read-only)"
	}
	invalidArguments := errors.New("invalid diagnostic command arguments; use --help for accepted flags")
	c := &cobra.Command{Use: name, Short: short, Args: func(_ *cobra.Command, args []string) error {
		if len(args) != 0 {
			return invalidArguments
		}
		return nil
	}, RunE: func(cmd *cobra.Command, _ []string) error {
		if format != "text" && format != "json" {
			return errors.New("format must be text or json")
		}
		report, e := controllerclient.Inspect(cmd.Context(), options, doctor)
		if e != nil {
			return e
		}
		if format == "json" {
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			e = enc.Encode(report)
		} else {
			_, e = fmt.Fprintf(cmd.OutOrStdout(), "Controller authority: %s\nPublished revision: %s\nObservation: %s\nAgent-reported state only; independent enforcement is not confirmed.\n", report.Authority, report.Revision, report.ObservedAt.Format(time.RFC3339Nano))
			if e == nil && report.Ready != nil {
				_, e = fmt.Fprintf(cmd.OutOrStdout(), "Controller ready: %t\n", *report.Ready)
			}
			if e == nil && len(report.Nodes) == 0 {
				_, e = fmt.Fprintln(cmd.OutOrStdout(), "No credentialed Agent scopes observed.")
			}
			for _, n := range report.Nodes {
				if e == nil {
					_, e = fmt.Fprintf(cmd.OutOrStdout(), "%s: %s (desired %s, reported %s; freshness remaining %dms)\n", n.AgentRef, n.State, n.DesiredRevision, n.ReportedRevision, n.FreshForMillis)
				}
			}
		}
		if e != nil {
			return errors.New("cannot write diagnostic report")
		}
		if doctor && !report.Healthy() {
			return errors.New("doctor found unready, missing, stale or incomplete Agent assertions")
		}
		return nil
	}}
	c.SetFlagErrorFunc(func(*cobra.Command, error) error { return invalidArguments })
	f := c.Flags()
	f.StringVar(&options.Origin, "controller", "", "Controller origin; HTTPS or literal loopback HTTP")
	f.StringVar(&options.KeyFile, "key-file", "", "Absolute private management-key file (not Agent or ingress Token)")
	f.StringVar(&options.CAFile, "ca-file", "", "Additional PEM CA file; normal certificate and hostname checks remain enabled")
	f.StringVar(&options.Agent, "agent", "", "Inspect one exact currently credentialed Agent scope")
	f.StringVar(&format, "format", "text", "Output text or json; JSON contains no policy, facts or credential values")
	f.DurationVar(&options.Timeout, "timeout", 5*time.Second, "Total diagnostic deadline, positive and at most 30s")
	_ = c.MarkFlagRequired("controller")
	_ = c.MarkFlagRequired("key-file")
	return c
}
