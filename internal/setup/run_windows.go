//go:build windows

// SPDX-License-Identifier: Apache-2.0
package setup

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

type pending struct {
	Recovery    Recovery `json:"recovery"`
	Original    uint32   `json:"original_code"`
	ResultKnown bool     `json:"native_result_known"`
}
type result struct {
	Native   *uint32 `json:"native_code"`
	Code     uint32  `json:"setup_code"`
	Recovery string  `json:"recovery"`
	Error    string  `json:"error,omitempty"`
	Logs     string  `json:"logs"`
}

func Report(message string) {
	proc := windows.NewLazySystemDLL("user32.dll").NewProc("MessageBoxW")
	if proc.Find() != nil {
		return
	}
	text, err := windows.UTF16PtrFromString(message)
	if err != nil {
		return
	}
	title, _ := windows.UTF16PtrFromString("insonic setup")
	proc.Call(0, uintptr(unsafe.Pointer(text)), uintptr(unsafe.Pointer(title)), 0x30)
}
func worker(c *cache, nonce string) error {
	var p pending
	if err := decodeFile(filepath.Join(c.root, "pending.json"), &p); err != nil {
		return err
	}
	if p.Recovery.Nonce != nonce || len(nonce) != 48 || p.Original == 0 {
		return errors.New("no matching failed upgrade recovery is pending")
	}
	if p.Recovery.Parent == 0 {
		return errors.New("recovery coordinator identity is missing")
	}
	// The parent holds the exclusive setup lock until this synchronous child
	// returns. The nonce binds this invocation to its private pending record.
	parent, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, p.Recovery.Parent)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(parent)
	var token windows.Token
	if err := windows.OpenProcessToken(parent, windows.TOKEN_QUERY, &token); err != nil {
		return err
	}
	defer token.Close()
	user, err := token.GetTokenUser()
	if err != nil {
		return err
	}
	own, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil || !user.User.Sid.Equals(own.User.Sid) || token.IsElevated() {
		return errors.New("recovery coordinator is not the ordinary invoking account")
	}
	source, err := c.source(p.Recovery.Before.Product)
	if err != nil {
		return err
	}
	if source != p.Recovery.Source {
		return errors.New("pending recovery source differs from private source record")
	}
	if _, err := native("MsiSetInternalUI", 2, 0); err != nil {
		return err
	}
	e := &nativeEngine{logDirectory: c.root}
	defer e.close()
	return Recover(e, p.Recovery)
}
func runWorker(c *cache, p pending) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	p.Recovery.Parent = uint32(os.Getpid())
	p.Recovery.Nonce, err = randomID()
	if err != nil {
		return err
	}
	if err := atomicJSON(filepath.Join(c.root, "pending.json"), p); err != nil {
		return err
	}
	workerLog := filepath.Join(c.root, "recovery-worker.log")
	if _, err := os.Lstat(workerLog); err == nil {
		if err := noReparse(workerLog); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	stream, err := os.OpenFile(workerLog, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer stream.Close()
	command := exec.Command(exe, "--recovery-worker", p.Recovery.Nonce)
	command.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: windows.CREATE_NO_WINDOW}
	command.Stdout = stream
	command.Stderr = stream
	// No context timeout or detached completion: never kill Windows Installer
	// or return while the critical restore is still running.
	return command.Run()
}
func previousSource(c *cache, before Choices, o Options) (Source, error) {
	if o.PreviousMSI != "" {
		path, err := filepath.Abs(o.PreviousMSI)
		if err != nil {
			return Source{}, err
		}
		input, err := pinFile(path)
		if err != nil {
			return Source{}, err
		}
		defer input.Close()
		p, err := inspectMSI(path)
		if err != nil {
			return Source{}, err
		}
		info, err := input.Stat()
		if err != nil {
			return Source{}, err
		}
		p.SHA256 = o.PreviousSHA
		p.Size = info.Size()
		if p.Product != before.Product || p.Version != before.Version {
			return Source{}, errors.New("selected previous MSI does not match the installed version")
		}
		if err := CopyPayload(io.Discard, input, p); err != nil {
			return Source{}, err
		}
		e := &nativeEngine{logDirectory: c.root}
		if err := e.ValidateInventory(Source{Package: p, Path: path}, before); err != nil {
			return Source{}, err
		}
		if _, err := input.Seek(0, io.SeekStart); err != nil {
			return Source{}, err
		}
		return c.retain(p, input, true)
	}
	source, err := c.source(before.Product)
	if err != nil {
		return Source{}, errors.New("previous full MSI is unavailable or corrupt; keep the working installation and retry with --previous-msi PATH --previous-sha256 SHA256 from that official release")
	}
	return source, nil
}
func nativeResult(code uint32) *uint32 { return &code }

func Run(args []string) int {
	o, err := Parse(args)
	quiet := false
	for _, a := range args {
		if a == "--quiet" || a == "--recovery-worker" {
			quiet = true
		}
	}
	var c *cache
	fail := func(err error) int {
		if c != nil && o.Worker == "" {
			_ = atomicJSON(filepath.Join(c.root, "last-result.json"), result{Native: nil, Code: 1603, Recovery: "not-started", Error: err.Error(), Logs: c.root})
		}
		fmt.Fprintln(os.Stderr, err)
		if !quiet {
			Report(err.Error())
		}
		return 1603
	}
	if err != nil {
		return fail(err)
	}
	if handled, code, err := limitedStart(args); handled {
		if err != nil {
			return fail(err)
		}
		return code
	}
	if o.Help {
		fmt.Fprintln(os.Stdout, Usage)
		if !o.Quiet {
			Report(Usage)
		}
		return 0
	}
	finish, err := apartment()
	if err != nil {
		return fail(err)
	}
	defer finish()
	c, err = openCache(o.Worker == "")
	if err != nil {
		return fail(err)
	}
	defer c.close()
	if o.Worker != "" {
		if err := worker(c, o.Worker); err != nil {
			return fail(err)
		}
		return 0
	}
	var outstanding pending
	pendingErr := decodeFile(filepath.Join(c.root, "pending.json"), &outstanding)
	if o.Resume {
		if pendingErr != nil {
			return fail(errors.New("no readable pending recovery is available"))
		}
		err = runWorker(c, outstanding)
		receipt := result{Code: outstanding.Original, Recovery: "completed", Logs: c.root}
		if outstanding.ResultKnown {
			receipt.Native = nativeResult(outstanding.Original)
		}
		if err != nil {
			receipt.Recovery = "incomplete"
			receipt.Error = err.Error()
		} else {
			err = c.clean(outstanding.Recovery.NewProduct)
			if err == nil {
				err = os.Remove(filepath.Join(c.root, "pending.json"))
			}
		}
		if err != nil {
			receipt.Code = 1603
			receipt.Error = err.Error()
		}
		if writeErr := atomicJSON(filepath.Join(c.root, "last-result.json"), receipt); writeErr != nil {
			receipt.Code = 1603
			receipt.Error = writeErr.Error()
		}
		_ = json.NewEncoder(os.Stdout).Encode(receipt)
		if receipt.Error != "" {
			message := fmt.Sprintf("Recovery requires attention; retained source and logs: %s", c.root)
			fmt.Fprintln(os.Stderr, message)
			if !o.Quiet {
				Report(message)
			}
		}
		return int(receipt.Code)
	}
	if !os.IsNotExist(pendingErr) {
		return fail(fmt.Errorf("a previous recovery record requires correction; run this setup with --resume-recovery and inspect %s", c.root))
	}
	exe, err := os.Executable()
	if err != nil {
		return fail(err)
	}
	input, err := pinFile(exe)
	if err != nil {
		return fail(err)
	}
	defer input.Close()
	info, err := input.Stat()
	if err != nil {
		return fail(err)
	}
	p, payload, err := ReadPayload(input, info.Size())
	if err != nil {
		return fail(err)
	}
	product, err := installedProduct()
	if err != nil {
		return fail(err)
	}
	if (o.Repair || o.Remove) && product != p.Product {
		return fail(errors.New("repair/removal requires setup for the installed product version"))
	}
	var prior *Choices
	var recovery pending
	oldEngine := &nativeEngine{logDirectory: c.root}
	defer oldEngine.close()
	if product != "" && product != p.Product {
		before, err := snapshot(product)
		if err != nil {
			return fail(err)
		}
		source, err := previousSource(c, before, o)
		if err != nil {
			return fail(err)
		}
		if err := oldEngine.ValidateSource(source); err != nil {
			return fail(fmt.Errorf("previous source failed validation before upgrade: %w", err))
		}
		if err := oldEngine.ValidateInventory(source, before); err != nil {
			return fail(err)
		}
		prior = &before
		recovery = pending{Recovery: Recovery{Source: source, Before: before, NewProduct: p.Product}, Original: 1603}
	}
	source, err := c.retain(p, payload)
	if err != nil {
		return fail(err)
	}
	e := &nativeEngine{logDirectory: c.root}
	defer e.close()
	if err := e.ValidateSource(source); err != nil {
		return fail(err)
	}
	properties, err := o.properties(prior, p)
	if err != nil {
		return fail(err)
	}
	if prior != nil {
		if err := atomicJSON(filepath.Join(c.root, "pending.json"), recovery); err != nil {
			return fail(err)
		}
	}
	ui := 5
	if o.Quiet {
		ui = 2
	}
	if _, err := native("MsiSetInternalUI", ui, 0); err != nil {
		return fail(err)
	}
	if err := e.logging("setup"); err != nil {
		return fail(err)
	}
	stopObserver, err := observeMSI(c.root, uint32(ui))
	if err != nil {
		return fail(err)
	}
	nativeCode, invokeErr := native("MsiInstallProductW", source.Path, properties)
	traceErr := stopObserver()
	if invokeErr != nil {
		nativeCode = 1603
	}
	e.close()
	oldEngine.close()
	receipt := result{Native: nativeResult(nativeCode), Code: nativeCode, Recovery: "not-needed", Logs: c.root}
	if traceErr != nil {
		receipt.Error = traceErr.Error()
	}
	// Windows Installer also reports committed success with a restart result.
	// Preserve it without attempting to roll back an already installed upgrade.
	committed := nativeCode == 0 || nativeCode == 3010 || nativeCode == 1641
	if !committed && prior != nil {
		recovery.Original = nativeCode
		recovery.ResultKnown = true
		if err := runWorker(c, recovery); err != nil {
			receipt.Recovery = "incomplete"
			receipt.Error = err.Error()
		} else {
			receipt.Recovery = "completed"
			if err := os.Remove(filepath.Join(c.root, "pending.json")); err != nil {
				receipt.Error = err.Error()
			}
			if e.State(p.Product) == -1 {
				if err := c.clean(p.Product); err != nil {
					receipt.Error = err.Error()
				}
			}
		}
	} else if committed {
		state := e.State(p.Product)
		if state != 5 && !(state == -1 && product == p.Product) {
			receipt.Code = 1603
			receipt.Error = "native success has unexpected product registration; recovery sources retained"
		}
		if prior != nil {
			if e.State(p.Product) != 5 || e.State(prior.Product) != -1 {
				receipt.Error = "successful native result has unexpected registration; recovery data retained"
				receipt.Code = 1603
			} else {
				if err := c.clean(prior.Product); err != nil {
					receipt.Error = err.Error()
				}
				if err := os.Remove(filepath.Join(c.root, "pending.json")); err != nil {
					receipt.Error = err.Error()
				}
			}
		}
		if receipt.Code == nativeCode && e.State(p.Product) == -1 {
			if err := c.clean(p.Product); err != nil {
				receipt.Error = err.Error()
			}
		}
	} else if e.State(p.Product) == -1 {
		if err := c.clean(p.Product); err != nil {
			receipt.Error = err.Error()
		}
	}
	if err := atomicJSON(filepath.Join(c.root, "last-result.json"), receipt); err != nil {
		receipt.Error = err.Error()
	}
	_ = json.NewEncoder(os.Stdout).Encode(receipt)
	if !o.Quiet && (receipt.Recovery != "not-needed" || receipt.Error != "") {
		message := "Setup did not complete. The previous installation was restored."
		if receipt.Recovery == "incomplete" {
			message = "Setup did not complete and recovery is incomplete. Retained local sources can be retried with --resume-recovery."
		} else if receipt.Error != "" {
			message = "Setup finished with a local recovery-data correction requiring attention."
		}
		Report(message + "\nDetails: " + filepath.Join(c.root, "last-result.json"))
	}
	return int(receipt.Code)
}

// CleanupUninstalled is called synchronously after a committed raw-MSI removal.
// A coordinator already holding the lock performs the same cleanup itself.
func CleanupUninstalled(product string) error {
	if !guidPattern.MatchString(product) || windows.GetCurrentProcessToken().IsElevated() {
		return errors.New("invalid ordinary-user cleanup context")
	}
	if (&nativeEngine{}).State(product) != -1 {
		return errors.New("product remains registered; retained source was preserved")
	}
	base, err := windows.KnownFolderPath(windows.FOLDERID_LocalAppData, 0)
	if err != nil {
		return err
	}
	if _, err := os.Lstat(filepath.Join(base, "insonic-setup")); os.IsNotExist(err) {
		return nil
	}
	c, err := openCache(true)
	if err != nil {
		return err
	}
	defer c.close()
	var p pending
	if err := decodeFile(filepath.Join(c.root, "pending.json"), &p); err == nil && p.Recovery.Before.Product == product {
		return nil
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}
	return c.clean(product)
}
