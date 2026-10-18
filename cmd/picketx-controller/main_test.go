// SPDX-License-Identifier: Apache-2.0
package main

import (
	"context"
	"io"
	"log/slog"
	"testing"
)

func TestHelpDoesNotStartOrFail(t *testing.T) {
	if e := run(context.Background(), []string{"-help"}, nil, slog.New(slog.NewTextHandler(io.Discard, nil))); e != nil {
		t.Fatal(e)
	}
}
func TestRejectUnexpectedArguments(t *testing.T) {
	for _, args := range [][]string{{"extra"}, {"-unknown"}} {
		if e := run(context.Background(), args, nil, slog.New(slog.NewTextHandler(io.Discard, nil))); e == nil {
			t.Fatal("unexpected argument accepted")
		}
	}
}
