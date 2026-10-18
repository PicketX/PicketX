// SPDX-License-Identifier: Apache-2.0
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/PicketX/PicketX/internal/controllerapp"
	"github.com/PicketX/PicketX/internal/runtimeconfig"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if e := run(ctx, os.Args[1:], os.Environ(), logger); e != nil {
		logger.Error("controller stopped", "error", e)
		os.Exit(1)
	}
}
func run(ctx context.Context, args, environment []string, logger *slog.Logger) error {
	fs := flag.NewFlagSet("picketx-controller", flag.ContinueOnError)
	path := fs.String("config", "", "runtime YAML configuration file (or PICKETX_CONFIG)")
	httpListen := fs.String("listen", "", "explicit HTTP listen override")
	agentListen := fs.String("agent-listen", "", "explicit Agent listen override")
	if e := fs.Parse(args); e != nil {
		if errors.Is(e, flag.ErrHelp) {
			return nil
		}
		return e
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments")
	}
	env := map[string]string{}
	for _, v := range environment {
		k, value, ok := strings.Cut(v, "=")
		if ok {
			env[k] = value
		}
	}
	overrides := map[string]string{}
	configExplicit := false
	fs.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "config":
			configExplicit = true
		case "listen":
			overrides["http.listen"] = *httpListen
		case "agent-listen":
			overrides["agent.listen"] = *agentListen
		}
	})
	if !configExplicit {
		*path = env["PICKETX_CONFIG"]
	}
	c, e := runtimeconfig.Load(*path, env, overrides)
	if e != nil {
		return e
	}
	return controllerapp.Run(ctx, c, logger, controllerapp.Dependencies{})
}
