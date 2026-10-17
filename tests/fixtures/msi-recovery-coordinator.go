//go:build ignore

// SPDX-License-Identifier: Apache-2.0
// CI-only ordering probe. This is not the product recovery coordinator.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"golang.org/x/sys/windows"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"unsafe"
)

func main() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	independent := false
	worker := false
	standalone := false
	fullUI := false
	var stage string
	var transaction uint64
	for _, arg := range os.Args[1:] {
		if arg == "--standalone" {
			standalone = true
			continue
		}
		if arg == "--full-ui" {
			fullUI = true
			continue
		}
		if arg == "--recovery-worker" {
			worker = true
			continue
		}
		if arg == "--independent" {
			independent = true
			continue
		}
		if value, err := strconv.ParseUint(arg, 0, 32); err == nil {
			transaction = value
			continue
		}
		if filepath.IsAbs(arg) && strings.HasPrefix(filepath.Base(arg), "insonic-ci-account-") {
			stage = arg
			continue
		}
	}
	if stage == "" || (!worker && !standalone && transaction == 0) || !strings.EqualFold(filepath.Dir(stage), os.Getenv("ProgramData")) {
		os.Exit(91)
	}
	marker, err := os.ReadFile(filepath.Join(stage, "owned-ci-account.json"))
	if err != nil {
		os.Exit(92)
	}
	var identity struct {
		SID string `json:"sid_sha256"`
	}
	if json.Unmarshal(marker, &identity) != nil {
		os.Exit(93)
	}
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		os.Exit(94)
	}
	hash := sha256.Sum256([]byte(user.User.Sid.String()))
	if hex.EncodeToString(hash[:]) != identity.SID || windows.GetCurrentProcessToken().IsElevated() {
		os.Exit(95)
	}
	var model struct {
		Old string `json:"old_product"`
		New string `json:"new_product"`
	}
	data, err := os.ReadFile(filepath.Join(stage, "ui-model.json"))
	if err != nil || json.Unmarshal(data, &model) != nil || model.Old == "" || model.New == "" {
		os.Exit(96)
	}
	proof := map[string]any{"feature_qualification": false, "pid": os.Getpid(), "args": os.Args, "user_sid_sha256": identity.SID, "phase": "startup"}
	receipt := "native/context.container-coordinator.json"
	if worker {
		receipt = "native/context.container-worker.json"
	}
	proof["recovery_worker"] = worker
	save := func() {
		data, _ := json.MarshalIndent(proof, "", "  ")
		_ = os.WriteFile(filepath.Join(stage, receipt), append(data, '\n'), 0600)
	}
	save()
	ole := windows.NewLazySystemDLL("ole32.dll")
	hr, _, _ := ole.NewProc("CoInitializeEx").Call(0, 2)
	proof["com_hresult"] = uint32(hr)
	save()
	if uint32(hr) != 0 && uint32(hr) != 1 {
		os.Exit(102)
	}
	dll := windows.NewLazySystemDLL("msi.dll")
	var pointers []*uint16
	defer func() { runtime.KeepAlive(pointers) }()
	ptr := func(value string) uintptr {
		p, err := windows.UTF16PtrFromString(value)
		if err != nil {
			os.Exit(97)
		}
		pointers = append(pointers, p)
		return uintptr(unsafe.Pointer(p))
	}
	query := func(product string) int32 {
		code, _, _ := dll.NewProc("MsiQueryProductStateW").Call(ptr(product))
		return int32(code)
	}
	properties := "ADDLOCAL=Core,CLI REMOVE=Desktop INSTALLDIR=\"" + filepath.Join(stage, "UI program é & spaces") + "\" START_MENU=0 DESKTOP_SHORTCUT=0 LICENSE_ACCEPTED=1 LAUNCH=0 REBOOT=ReallySuppress"
	if worker {
		source := filepath.Join(stage, "ui-old.msi")
		info, err := os.Lstat(source)
		if err != nil || !info.Mode().IsRegular() || query(model.Old) != 1 {
			proof["source_or_state_invalid"] = true
			save()
			os.Exit(98)
		}
		dll.NewProc("MsiSetInternalUI").Call(2, 0)
		dll.NewProc("MsiEnableLogW").Call(0x1000, ptr(filepath.Join(stage, "native/context.container-worker-remove.log")), 0)
		code, _, _ := dll.NewProc("MsiConfigureProductExW").Call(ptr(model.Old), 0, 2, ptr("REMOVE=ALL LAUNCH=0 REBOOT=ReallySuppress"))
		proof["remove_code"] = uint32(code)
		proof["after_remove"] = query(model.Old)
		save()
		if code == 0 && query(model.Old) == -1 {
			dll.NewProc("MsiEnableLogW").Call(0x1000, ptr(filepath.Join(stage, "native/context.container-worker-install.log")), 0)
			code, _, _ = dll.NewProc("MsiInstallProductW").Call(ptr(source), ptr(properties))
			proof["install_code"] = uint32(code)
		}
		dll.NewProc("MsiEnableLogW").Call(0, 0, 0)
		proof["old_final"] = query(model.Old)
		proof["new_final"] = query(model.New)
		proof["phase"] = "completed"
		save()
		if code != 0 || query(model.Old) != 5 || query(model.New) != -1 {
			os.Exit(99)
		}
		os.Exit(0)
	}
	var code uintptr
	if !standalone {
		var event windows.Handle
		code, _, _ = dll.NewProc("MsiJoinTransaction").Call(uintptr(transaction), 0, uintptr(unsafe.Pointer(&event)))
		proof["join_code"] = uint32(code)
		save()
		if code != 0 {
			os.Exit(int(code))
		}
		if event != 0 {
			defer windows.CloseHandle(event)
		}
	}

	if independent {
		code, _, _ = dll.NewProc("MsiEndTransaction").Call(1)
		proof["empty_container_commit_code"] = uint32(code)
		save()
		if code != 0 {
			os.Exit(int(code))
		}
	}
	ui := uintptr(2)
	if fullUI {
		ui = 5
		properties = strings.Replace(properties, " LICENSE_ACCEPTED=1", "", 1)
	}
	dll.NewProc("MsiSetInternalUI").Call(ui, 0)
	level, _, _ := dll.NewProc("MsiSetInternalUI").Call(0, 0)
	proof["configured_ui_level"] = uint32(level)
	save()
	dll.NewProc("MsiEnableLogW").Call(0x1000, ptr(filepath.Join(stage, "native/context.container-inner.log")), 0)

	proof["phase"] = "before-inner"
	save()
	original, _, _ := dll.NewProc("MsiInstallProductW").Call(ptr(filepath.Join(stage, "ui-bad.msi")), ptr(properties))
	proof["inner_code"] = uint32(original)
	proof["phase"] = "before-end"
	save()
	state := uintptr(1)
	if original != 0 {
		state = 0
	}
	if !independent && !standalone {
		code, _, _ = dll.NewProc("MsiEndTransaction").Call(state)
		proof["end_code"] = uint32(code)
	}
	proof["old_after_end"] = query(model.Old)
	proof["new_after_end"] = query(model.New)
	proof["phase"] = "after-end"
	save()
	if original != 0 && query(model.Old) == 1 {
		exe, err := os.Executable()
		if err != nil {
			os.Exit(100)
		}
		stream, err := os.Create(filepath.Join(stage, "native/context.container-fresh-worker-stream.log"))
		if err != nil {
			os.Exit(101)
		}
		command := exec.Command(exe, "--recovery-worker", stage)
		command.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
		command.Stdout = stream
		command.Stderr = stream
		err = command.Run()
		_ = stream.Close()
		result := 0
		if err != nil {
			result = 1
			if failed, ok := err.(*exec.ExitError); ok {
				result = failed.ExitCode()
			}
		}
		proof["fresh_recovery_worker_code"] = result
		save()
	}
	dll.NewProc("MsiEnableLogW").Call(0, 0, 0)
	proof["old_final"] = query(model.Old)
	proof["new_final"] = query(model.New)
	proof["phase"] = "completed"
	save()
	os.Exit(int(original))
}
