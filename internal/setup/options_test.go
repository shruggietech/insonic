// SPDX-License-Identifier: Apache-2.0
package setup

import (
	"strings"
	"testing"
)

func TestQuietUpgradeInheritsPriorChoicesAndAcceptsOnlyExplicitOverrides(t *testing.T) {
	o, err := Parse([]string{"--quiet", "--accept-license"})
	if err != nil {
		t.Fatal(err)
	}
	p, err := o.properties(&testChoices, testPackage)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"ADDLOCAL=Core,CLI", "REMOVE=Desktop", "START_MENU=0", "DESKTOP_SHORTCUT=0", "LAUNCH=0", "OPEN_DOCUMENTATION=0", "LICENSE_ACCEPTED=1"} {
		if !strings.Contains(p, want) {
			t.Fatal(p, want)
		}
	}
	o, err = Parse([]string{"--quiet", "--accept-license", "--clients", "desktop", "--start-menu", "1"})
	if err != nil {
		t.Fatal(err)
	}
	p, err = o.properties(&testChoices, testPackage)
	if err != nil || !strings.Contains(p, "ADDLOCAL=Core,Desktop") || !strings.Contains(p, "REMOVE=CLI") || !strings.Contains(p, "START_MENU=1") {
		t.Fatal(p, err)
	}
}
func TestInteractiveFreshSetupPreservesUncheckedLicenseAndMSILaunchDefaults(t *testing.T) {
	o, err := Parse(nil)
	if err != nil {
		t.Fatal(err)
	}
	p, err := o.properties(nil, testPackage)
	if err != nil {
		t.Fatal(err)
	}
	for _, excluded := range []string{"LICENSE_ACCEPTED", "LAUNCH=", "OPEN_DOCUMENTATION="} {
		if strings.Contains(p, excluded) {
			t.Fatal(p)
		}
	}
}
func TestSetupRejectsUnboundedMSIOptionsAndUnpairedLegacySource(t *testing.T) {
	for _, args := range [][]string{{"ALLUSERS=1"}, {"--accept-license"}, {"--quiet", "--quiet"}, {"--clients", "none"}, {"--previous-msi", "old.msi"}, {"--previous-sha256", strings.Repeat("a", 64)}, {"--install-dir", "C:\\x\" REMOVE=ALL"}, {"--repair", "--remove"}, {"--recovery-worker", strings.Repeat("a", 48), "--quiet"}} {
		if _, err := Parse(args); err == nil {
			t.Fatal(args)
		}
	}
}
