// SPDX-License-Identifier: Apache-2.0
package setup

import (
	"errors"
	"fmt"
	"strings"
)

type File struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size_bytes"`
}
type Choices struct {
	Product         string `json:"product_code"`
	Version         string `json:"version"`
	Directory       string `json:"directory"`
	Desktop         bool   `json:"desktop"`
	CLI             bool   `json:"cli"`
	StartMenu       bool   `json:"start_menu"`
	DesktopShortcut bool   `json:"desktop_shortcut"`
	Files           []File `json:"files"`
}
type Source struct {
	Package Package `json:"package"`
	Path    string  `json:"path"`
}
type Recovery struct {
	Source     Source  `json:"source"`
	Before     Choices `json:"before"`
	NewProduct string  `json:"new_product"`
	Nonce      string  `json:"nonce"`
	Parent     uint32  `json:"parent_pid"`
}

func property(name, value string) (string, error) {
	if strings.ContainsAny(value, "\x00\r\n\"") || len(value) > 30000 {
		return "", errors.New("invalid MSI property value")
	}
	return name + `="` + value + `"`, nil
}
func choiceProperties(c Choices) (string, error) {
	if !guidPattern.MatchString(c.Product) || !versionPattern.MatchString(c.Version) || (!c.Desktop && !c.CLI) || c.Directory == "" {
		return "", errors.New("previous installation choices are incomplete")
	}
	directory, err := property("INSTALLDIR", c.Directory)
	if err != nil {
		return "", err
	}
	features := []string{"Core"}
	removed := []string{}
	for _, f := range []struct {
		name     string
		selected bool
	}{{"Desktop", c.Desktop}, {"CLI", c.CLI}} {
		if f.selected {
			features = append(features, f.name)
		} else {
			removed = append(removed, f.name)
		}
	}
	properties := []string{directory, "ADDLOCAL=" + strings.Join(features, ","), fmt.Sprintf("START_MENU=%d DESKTOP_SHORTCUT=%d", bit(c.StartMenu), bit(c.DesktopShortcut))}
	if len(removed) > 0 {
		properties = append(properties, "REMOVE="+strings.Join(removed, ","))
	}
	return strings.Join(properties, " "), nil
}
func bit(v bool) int {
	if v {
		return 1
	}
	return 0
}
func RecoveryProperties(c Choices) (string, error) {
	p, err := choiceProperties(c)
	if err != nil {
		return "", err
	}
	return p + " LICENSE_ACCEPTED=1 LAUNCH=0 OPEN_DOCUMENTATION=0 REBOOT=ReallySuppress", nil
}

type RecoveryEngine interface {
	State(string) int32
	ValidateSource(Source) error
	Remove(string) uint32
	Install(Source, string) uint32
	Verify(Choices) error
}

// Recover runs in a fresh, ordinary-user process after the original MSI returns.
// A valid source is pinned before any removal. Never treat advertised state or
// restored files alone as an installed product, and never remove a new product
// left installed by an unexpected engine result.
func Recover(engine RecoveryEngine, r Recovery) error {
	if err := r.Source.Package.Validate(); err != nil {
		return err
	}
	if r.Before.Product != r.Source.Package.Product || r.Before.Version != r.Source.Package.Version || !guidPattern.MatchString(r.NewProduct) || r.NewProduct == r.Before.Product {
		return errors.New("recovery product identities differ")
	}
	properties, err := RecoveryProperties(r.Before)
	if err != nil {
		return err
	}
	if err = engine.ValidateSource(r.Source); err != nil {
		return err
	}
	if engine.State(r.NewProduct) != -1 {
		return errors.New("new product registration remains; recovery retained for explicit correction")
	}
	state := engine.State(r.Before.Product)
	if state == 5 {
		return engine.Verify(r.Before)
	}
	if state != 1 && state != -1 {
		return errors.New("unexpected old product state; no product was removed")
	}
	if state == 1 {
		if result := engine.Remove(r.Before.Product); result != 0 {
			return fmt.Errorf("old advertised registration removal returned %d", result)
		}
		if engine.State(r.Before.Product) != -1 {
			return errors.New("old advertised registration remains")
		}
	}
	if result := engine.Install(r.Source, properties); result != 0 {
		return fmt.Errorf("previous full-package reinstallation returned %d", result)
	}
	if engine.State(r.Before.Product) != 5 || engine.State(r.NewProduct) != -1 {
		return errors.New("previous installed registration was not restored")
	}
	return engine.Verify(r.Before)
}
