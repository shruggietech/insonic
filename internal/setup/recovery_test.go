// SPDX-License-Identifier: Apache-2.0
package setup

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

var testPackage = Package{Product: "{11111111-1111-5111-8111-111111111111}", Upgrade: Family, Version: "1.0.0", SHA256: strings.Repeat("a", 64), Size: 100}
var testChoices = Choices{Product: testPackage.Product, Version: "1.0.0", Directory: `C:\Users\ordinary\Program é & space`, CLI: true}

type fakeRecovery struct {
	states                   map[string]int32
	calls                    []string
	sourceError, verifyError error
}

func (f *fakeRecovery) State(product string) int32 { return f.states[product] }
func (f *fakeRecovery) ValidateSource(Source) error {
	f.calls = append(f.calls, "validate")
	return f.sourceError
}
func (f *fakeRecovery) Remove(product string) uint32 {
	f.calls = append(f.calls, "remove")
	f.states[product] = -1
	return 0
}
func (f *fakeRecovery) Install(source Source, properties string) uint32 {
	f.calls = append(f.calls, "install:"+properties)
	f.states[source.Package.Product] = 5
	return 0
}
func (f *fakeRecovery) Verify(Choices) error {
	f.calls = append(f.calls, "verify")
	return f.verifyError
}
func recoveryFixture() (Recovery, *fakeRecovery) {
	newProduct := "{22222222-2222-5222-8222-222222222222}"
	return Recovery{Source: Source{Package: testPackage, Path: `C:\private\source.msi`}, Before: testChoices, NewProduct: newProduct}, &fakeRecovery{states: map[string]int32{testPackage.Product: 1, newProduct: -1}}
}
func TestRecoveryValidatesBeforeDestructiveRemovalAndRestoresPriorChoices(t *testing.T) {
	r, engine := recoveryFixture()
	if err := Recover(engine, r); err != nil {
		t.Fatal(err)
	}
	if len(engine.calls) != 4 || engine.calls[0] != "validate" || engine.calls[1] != "remove" || engine.calls[3] != "verify" {
		t.Fatal(engine.calls)
	}
	properties := engine.calls[2]
	for _, want := range []string{`ADDLOCAL=Core,CLI`, `REMOVE=Desktop`, `INSTALLDIR="C:\Users\ordinary\Program é & space"`, `START_MENU=0`, `DESKTOP_SHORTCUT=0`, `LICENSE_ACCEPTED=1`, `LAUNCH=0`, `OPEN_DOCUMENTATION=0`} {
		if !strings.Contains(properties, want) {
			t.Fatal(properties, want)
		}
	}
}
func TestRecoveryRefusesMissingCorruptOrForeignSourceBeforeRemoval(t *testing.T) {
	for _, change := range []func(*Recovery, *fakeRecovery){
		func(r *Recovery, f *fakeRecovery) { f.sourceError = errors.New("missing source") },
		func(r *Recovery, f *fakeRecovery) {
			r.Source.Package.Upgrade = "{33333333-3333-5333-8333-333333333333}"
		},
		func(r *Recovery, f *fakeRecovery) { r.Before.Product = r.NewProduct },
		func(r *Recovery, f *fakeRecovery) { f.states[r.NewProduct] = 5 },
	} {
		r, f := recoveryFixture()
		change(&r, f)
		if Recover(f, r) == nil {
			t.Fatal("accepted invalid recovery")
		}
		for _, call := range f.calls {
			if call == "remove" || strings.HasPrefix(call, "install:") {
				t.Fatal(f.calls)
			}
		}
	}
}
func TestRecoveryAcceptsAlreadyRestoredProductOnlyAfterVerifyingChoicesAndBytes(t *testing.T) {
	r, f := recoveryFixture()
	f.states[r.Before.Product] = 5
	f.verifyError = errors.New("wrong payload")
	if Recover(f, r) == nil {
		t.Fatal("claimed restored from registration alone")
	}
	if !reflect.DeepEqual(f.calls, []string{"validate", "verify"}) {
		t.Fatal(f.calls)
	}
}
func TestRecoveryReinstallsAbsentOldProductWithoutRemovingAnotherProduct(t *testing.T) {
	r, f := recoveryFixture()
	f.states[r.Before.Product] = -1
	if err := Recover(f, r); err != nil {
		t.Fatal(err)
	}
	for _, call := range f.calls {
		if call == "remove" {
			t.Fatal(f.calls)
		}
	}
}
func TestMSIPropertiesRejectCommandInjectionAndPreserveExactDirectory(t *testing.T) {
	for _, dir := range []string{"C:\\valid\" REMOVE=ALL", "C:\\bad\nLAUNCH=1", "C:\\bad\x00"} {
		c := testChoices
		c.Directory = dir
		if _, err := RecoveryProperties(c); err == nil {
			t.Fatal("accepted", dir)
		}
	}
}
