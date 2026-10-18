// SPDX-License-Identifier: Apache-2.0
package controllerapp

import (
	"path/filepath"
	"syscall"
	"testing"
)

func TestFIFOInputRejectedWithoutBlocking(t *testing.T) {
	p := filepath.Join(t.TempDir(), "fifo")
	if e := syscall.Mkfifo(p, 0600); e != nil {
		t.Fatal(e)
	}
	if _, e := readFile(p, false); e == nil {
		t.Fatal("FIFO accepted")
	}
}
