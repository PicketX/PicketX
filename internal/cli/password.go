// SPDX-License-Identifier: Apache-2.0
package cli

import (
	"fmt"
	"github.com/PicketX/PicketX/internal/password"
	"github.com/spf13/cobra"
	"io"
	"strings"
)

func passwordCommand() *cobra.Command {
	return &cobra.Command{Use: "password-hash", Short: "Hash a local-human password from stdin (12–72 UTF-8 bytes)", Args: func(_ *cobra.Command, args []string) error {
		if len(args) != 0 {
			return fmt.Errorf("password-hash accepts no arguments; use stdin")
		}
		return nil
	}, RunE: func(cmd *cobra.Command, _ []string) error {
		raw, e := io.ReadAll(io.LimitReader(cmd.InOrStdin(), 75))
		if e != nil || len(raw) > 74 {
			return fmt.Errorf("invalid password input")
		}
		secret := strings.TrimSuffix(string(raw), "\n")
		if len(secret) < len(raw) {
			secret = strings.TrimSuffix(secret, "\r")
		}
		if strings.ContainsAny(secret, "\r\n") {
			return fmt.Errorf("password must be one line")
		}
		hash, e := password.Hash(secret)
		if e != nil {
			return e
		}
		_, e = fmt.Fprintln(cmd.OutOrStdout(), hash)
		return e
	}}
}
