// SPDX-License-Identifier: Apache-2.0
package speakerdiscovery

// IndependentContext resolves names without borrowing manual or acoustic
// assignments. In particular, acoustic disambiguation cannot enroll itself.
func IndependentContext(c Context, h []Hypothesis, p Policy) ([]Decision, error) {
	c.Owners = map[string]string{}
	c.OwnerEvidence = nil
	c.OwnerIdentities = nil
	c.Acoustic = AcousticEvidence{Availability: "none"}
	return Resolve(c, h, p)
}
