// SPDX-License-Identifier: Apache-2.0
package configuration

import "github.com/PicketX/PicketX/internal/strictjson"

// DecodeFacts requires exact field spellings, rejecting nulls and duplicate keys.
func DecodeFacts(data []byte) (Facts, error) {
	var f Facts
	e := strictjson.Decode(data, &f)
	return f, e
}
