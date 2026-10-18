// SPDX-License-Identifier: Apache-2.0
package runtimeconfig

import (
	"path/filepath"
	"syscall"
	"testing"
)

func TestFIFOConfigRejectedWithoutBlocking(t *testing.T) {
	p := filepath.Join(t.TempDir(), "fifo")
	if e := syscall.Mkfifo(p, 0600); e != nil {
		t.Fatal(e)
	}
	if _, e := Load(p, nil, nil); e == nil {
		t.Fatal("FIFO accepted")
	}
}
