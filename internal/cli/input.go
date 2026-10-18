// SPDX-License-Identifier: Apache-2.0
package cli

import (
	"fmt"
	"github.com/PicketX/PicketX/pkg/configuration"
	"io"
	"os"
)

func readBounded(path string) ([]byte, error) {
	info, e := os.Stat(path)
	if e != nil {
		return nil, e
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("input must be a regular file")
	}
	f, e := os.Open(path)
	if e != nil {
		return nil, e
	}
	defer f.Close()
	data, e := io.ReadAll(io.LimitReader(f, configuration.MaxBytes+1))
	if e != nil {
		return nil, e
	}
	if len(data) > configuration.MaxBytes {
		return nil, fmt.Errorf("input exceeds %d bytes", configuration.MaxBytes)
	}
	return data, nil
}
func decodeFacts(data []byte) (configuration.Facts, error) { return configuration.DecodeFacts(data) }
