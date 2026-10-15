// SPDX-License-Identifier: Apache-2.0
package main

import (
	"fmt"
	"github.com/PicketX/PicketX/internal/cli"
	"os"
)

func main() {
	if err := cli.New(os.Stdout, os.Stderr).Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
