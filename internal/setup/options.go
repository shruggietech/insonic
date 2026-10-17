// SPDX-License-Identifier: Apache-2.0
package setup

import (
	"errors"
	"strings"
)

const Usage = "insonic setup: open normally for native setup pages, or use --quiet --accept-license. Options: --install-dir PATH, --clients both|desktop|cli, --start-menu 0|1, --desktop-shortcut 0|1, --repair, --remove, --previous-msi PATH --previous-sha256 SHA256, --resume-recovery. Full UI starts with license acceptance unchecked. Recovery sources and logs are retained privately under LocalAppData/ insonic-setup."

type Options struct {
	Quiet, AcceptLicense, Repair, Remove, Help, Resume   bool
	Directory, Clients, PreviousMSI, PreviousSHA, Worker string
	StartMenu, DesktopShortcut                           *bool
}

func Parse(args []string) (Options, error) {
	var o Options
	seen := map[string]bool{}
	for i := 0; i < len(args); i++ {
		a := args[i]
		if seen[a] {
			return o, errors.New("duplicate setup option")
		}
		seen[a] = true
		switch a {
		case "--quiet":
			o.Quiet = true
		case "--accept-license":
			o.AcceptLicense = true
		case "--repair":
			o.Repair = true
		case "--remove":
			o.Remove = true
		case "--help":
			o.Help = true
		case "--resume-recovery":
			o.Resume = true
		case "--install-dir", "--clients", "--previous-msi", "--previous-sha256", "--start-menu", "--desktop-shortcut", "--recovery-worker":
			i++
			if i >= len(args) || args[i] == "" || strings.HasPrefix(args[i], "--") {
				return o, errors.New("setup option requires a value")
			}
			value := args[i]
			if strings.ContainsAny(value, "\x00\r\n\"") {
				return o, errors.New("invalid setup option value")
			}
			switch a {
			case "--install-dir":
				o.Directory = value
			case "--clients":
				o.Clients = value
			case "--previous-msi":
				o.PreviousMSI = value
			case "--previous-sha256":
				o.PreviousSHA = strings.ToLower(value)
			case "--recovery-worker":
				o.Worker = value
			default:
				if value != "0" && value != "1" {
					return o, errors.New("shortcut options require 0 or 1")
				}
				b := value == "1"
				if a == "--start-menu" {
					o.StartMenu = &b
				} else {
					o.DesktopShortcut = &b
				}
			}
		default:
			return o, errors.New(Usage)
		}
	}
	if o.Repair && o.Remove || o.AcceptLicense && !o.Quiet || o.Clients != "" && o.Clients != "both" && o.Clients != "desktop" && o.Clients != "cli" || (o.PreviousMSI == "") != (o.PreviousSHA == "") || o.PreviousSHA != "" && !digestPattern.MatchString(o.PreviousSHA) {
		return o, errors.New("invalid setup options; use --help")
	}
	if (o.Worker != "" && (len(args) != 2 || len(o.Worker) != 48)) || (o.Resume && len(args) != 1) {
		return o, errors.New("recovery must run separately from other setup operations")
	}
	return o, nil
}
func (o Options) properties(prior *Choices, current Package) (string, error) {
	var values []string
	if prior != nil && o.Quiet {
		copy := *prior
		if o.Clients != "" {
			copy.Desktop = o.Clients != "cli"
			copy.CLI = o.Clients != "desktop"
		}
		if o.Directory != "" {
			copy.Directory = o.Directory
		}
		if o.StartMenu != nil {
			copy.StartMenu = *o.StartMenu
		}
		if o.DesktopShortcut != nil {
			copy.DesktopShortcut = *o.DesktopShortcut
		}
		p, err := choiceProperties(copy)
		if err != nil {
			return "", err
		}
		values = append(values, p)
	} else {
		if o.Directory != "" {
			p, err := property("INSTALLDIR", o.Directory)
			if err != nil {
				return "", err
			}
			values = append(values, p)
		}
		if o.Clients != "" {
			features := "Core"
			removed := ""
			if o.Clients != "cli" {
				features += ",Desktop"
			} else {
				removed = "Desktop"
			}
			if o.Clients != "desktop" {
				features += ",CLI"
			} else {
				removed = "CLI"
			}
			values = append(values, "ADDLOCAL="+features)
			if removed != "" {
				values = append(values, "REMOVE="+removed)
			}
		}
		if o.StartMenu != nil {
			values = append(values, "START_MENU="+string(rune('0'+bit(*o.StartMenu))))
		}
		if o.DesktopShortcut != nil {
			values = append(values, "DESKTOP_SHORTCUT="+string(rune('0'+bit(*o.DesktopShortcut))))
		}
	}
	if o.Repair {
		values = append(values, "REINSTALL=ALL REINSTALLMODE=omus")
	}
	if o.Remove {
		values = []string{"REMOVE=ALL"}
	}
	if o.AcceptLicense {
		values = append(values, "LICENSE_ACCEPTED=1")
	}
	if o.Quiet || o.Repair || o.Remove || prior != nil {
		values = append(values, "LAUNCH=0 OPEN_DOCUMENTATION=0")
	}
	values = append(values, "REBOOT=ReallySuppress")
	return strings.Join(values, " "), nil
}
