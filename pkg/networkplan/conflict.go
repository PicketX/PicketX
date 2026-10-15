// SPDX-License-Identifier: Apache-2.0
package networkplan

import "fmt"

// Claims are compared independently of active source membership. Trusted network
// space is supplied by the resolver; context names and labels prove no isolation.
func (p *Plan) validateConflicts() error {
	var targets []Protection
	for _, key := range p.targetKeys() {
		current := p.targets[key].Protection
		for _, previous := range targets {
			if current.Key.AgentRef != previous.Key.AgentRef || current.Context.NetworkSpaceID != previous.Context.NetworkSpaceID {
				continue
			}
			for _, a := range current.Matches {
				for _, b := range previous.Matches {
					if a.Protocol == b.Protocol && a.Port == b.Port && a.Destination.Overlaps(b.Destination) {
						return fmt.Errorf("physical conflict on Agent %q space %q: %s/%s context %q Bindings %v overlaps %s/%s context %q Bindings %v; distinct context IDs cannot prove disjointness", key.AgentRef, current.Context.NetworkSpaceID, key.Entry.ResourceName, key.Entry.EndpointName, key.ContextID, current.BindingIDs, previous.Key.Entry.ResourceName, previous.Key.Entry.EndpointName, previous.Key.ContextID, previous.BindingIDs)
					}
				}
			}
		}
		targets = append(targets, current)
	}
	return nil
}
