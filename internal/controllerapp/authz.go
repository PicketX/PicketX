// SPDX-License-Identifier: Apache-2.0
package controllerapp

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
)

// ownedSocket only unlinks the exact object it created. Never delete a socket or
// regular file subsequently installed by another process at the same path.
type ownedSocket struct {
	net.Listener
	path string
	info os.FileInfo
}

func (l *ownedSocket) Close() error {
	e := l.Listener.Close()
	if info, err := os.Lstat(l.path); err == nil && os.SameFile(info, l.info) {
		if err = os.Remove(l.path); err != nil && e == nil {
			e = err
		}
	}
	return e
}
func listenAuthorization(network, address string, listen func(string, string) (net.Listener, error)) (net.Listener, error) {
	if network != "unix" {
		return listen(network, address)
	}
	dir, e := os.Lstat(filepath.Dir(address))
	if e != nil || !dir.IsDir() || dir.Mode().Perm()&0077 != 0 {
		return nil, fmt.Errorf("authorization socket requires existing private directory")
	}
	if _, e = os.Lstat(address); !os.IsNotExist(e) {
		return nil, fmt.Errorf("authorization socket path must not exist")
	}
	l, e := listen(network, address)
	if e != nil {
		return nil, e
	}
	if unix, ok := l.(*net.UnixListener); ok {
		unix.SetUnlinkOnClose(false)
	}
	info, e := os.Lstat(address)
	if e != nil {
		l.Close()
		return nil, e
	}
	owned := &ownedSocket{Listener: l, path: address, info: info}
	if e = os.Chmod(address, 0600); e != nil {
		owned.Close()
		return nil, e
	}
	return owned, nil
}
