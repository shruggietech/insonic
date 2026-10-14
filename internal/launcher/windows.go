//go:build windows

// SPDX-License-Identifier: Apache-2.0
package launcher

import (
	"errors"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

func SystemDirectory() (string, error) { return windows.GetSystemDirectory() }
func Launch(p Plan) error {
	if handled, e := launchLimited(p); handled || e != nil {
		return e
	}
	cmd := exec.Command(p.Executable)
	cmd.Dir = filepath.Dir(p.Environment["INSONIC_LAUNCH_CLI"])
	if p.Mode == "desktop" {
		cmd.Dir = filepath.Dir(p.Executable)
		cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: windows.CREATE_NO_WINDOW}
	} else {
		// A visible persistent terminal is an explicitly elected human launcher action.
		// Automated checks only use --describe/--preflight and never call this path.
		cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: windows.CREATE_NEW_CONSOLE, CmdLine: p.CommandLine}
		cmd.Env = os.Environ()
		for k, v := range p.Environment {
			cmd.Env = append(cmd.Env, k+"="+v)
		}
	}
	return cmd.Start()
}
func NoActiveClients(directory string) error {
	physical, e := canonicalPath(directory)
	if e != nil {
		return errors.New("cannot resolve the program directory; correct its aliases before maintenance")
	}
	snapshot, e := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if e != nil {
		return errors.New("cannot inspect active clients; close the application before maintenance")
	}
	defer windows.CloseHandle(snapshot)
	var entry windows.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))
	e = windows.Process32First(snapshot, &entry)
	for e == nil {
		if entry.ProcessID != uint32(os.Getpid()) {
			h, er := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, entry.ProcessID)
			if er == nil {
				buf := make([]uint16, 32768)
				n := uint32(len(buf))
				er = windows.QueryFullProcessImageName(h, 0, &buf[0], &n)
				windows.CloseHandle(h)
				if er == nil {
					image := windows.UTF16ToString(buf[:n])
					resolved, er := canonicalProcessImage(image)
					if er != nil && (containsPath(physical, image) || containsPath(directory, image)) {
						return errors.New("cannot resolve an active process path; finish work, close clients and retry maintenance")
					}
					if er == nil && containsPath(physical, resolved) {
						return errors.New("active clients or runtime use this installation; finish work, close them and retry maintenance")
					}
				}
			}
		}
		e = windows.Process32Next(snapshot, &entry)
	}
	if e != windows.ERROR_NO_MORE_FILES {
		return errors.New("active-client inspection failed; retry after closing clients")
	}
	return nil
}

// Installer registration is the repair authority if an owned marker is damaged.
func registeredInstallation(directory string) bool {
	k, e := registry.OpenKey(registry.CURRENT_USER, `Software\ShruggieTech\insonic`, registry.QUERY_VALUE|registry.WOW64_64KEY)
	if e != nil {
		return false
	}
	defer k.Close()
	saved, _, e := k.GetStringValue("InstallDir")
	if e != nil || !filepath.IsAbs(saved) {
		return false
	}
	physical, e := canonicalPath(directory)
	if e != nil {
		return false
	}
	registered, e := canonicalPath(saved)
	return e == nil && strings.EqualFold(physical, registered)
}
