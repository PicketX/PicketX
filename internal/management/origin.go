// SPDX-License-Identifier: Apache-2.0
package management

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
)

// ValidateBrowserOrigin requires an exact public origin, not an arbitrary URL.
func ValidateBrowserOrigin(value string) error {
	u, e := url.Parse(value)
	if e != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.Path != "" || u.Opaque != "" || u.String() != value {
		return fmt.Errorf("browser origin must be exact scheme and host")
	}
	ip := net.ParseIP(u.Hostname())
	if u.Scheme != "https" && !(u.Scheme == "http" && (u.Hostname() == "localhost" || ip != nil && ip.IsLoopback())) {
		return fmt.Errorf("browser requires HTTPS except loopback development")
	}
	if u.Port() != "" {
		p, e := strconv.Atoi(u.Port())
		if e != nil || p < 1 || p > 65535 {
			return fmt.Errorf("invalid browser origin port")
		}
	}
	return nil
}
