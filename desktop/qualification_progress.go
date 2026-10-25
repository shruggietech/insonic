// SPDX-License-Identifier: Apache-2.0
package desktop

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"sync"
	"time"

	"github.com/shruggietech/insonic/internal/qualificationtiming"
)

// This observer is created only by explicit disposable qualification. Its clock
// is native monotonic elapsed time; a receipt is not proof that JS consumed the
// receipt RPC's own return value. Fixed stages never contain paths or input data.
type qualificationProgress struct {
	mu           sync.Mutex
	origin       time.Time
	clock        func() time.Time
	out          io.Writer
	mode         string
	count        int
	last         time.Duration
	finished     bool
	frontendSeen bool
	frontendLast float64
}

func newQualificationProgress(mode string, out io.Writer, clock func() time.Time) *qualificationProgress {
	return &qualificationProgress{origin: clock(), clock: clock, out: out, mode: mode}
}

func validQualificationStep(step string) bool {
	switch step {
	case "module", "fixtures", "mounted", "ready", "library-import", "audio-playback", "video-playback", "metadata-date", "assembly", "cue-seek", "terms", "speakers", "pipelines", "jobs", "settings", "keyboard-help", "explore-calendar", "explore-query", "explore-graph", "query-assistance", "rosters", "audio-replacement", "model-references", "import-presets", "complete",
		"journey-entered", "journey-fixtures-request", "journey-fixtures-returned", "help-open-request", "help-open-returned", "help-control-ready", "help-frame-ready", "help-fonts-ready", "help-network-blocked", "help-reference-dispatched", "help-navigation-complete", "help-closed", "help-journey-complete", "complete-smoke-request", "watchdog-timeout":
		return true
	}
	return false
}

func (p *qualificationProgress) emit(value map[string]any) {
	value["kind"] = "desktop-qualification-progress"
	value["mode"] = p.mode
	value["observation"] = "native-receipt"
	value["elapsed_ns"] = p.last.Nanoseconds()
	raw, _ := json.Marshal(value)
	fmt.Fprintln(p.out, string(raw))
}

func (p *qualificationProgress) record(stage string) bool {
	return p.recordObservation(stage, nil)
}

func (p *qualificationProgress) recordFrontend(stage string, elapsedMilliseconds float64) bool {
	if math.IsNaN(elapsedMilliseconds) || math.IsInf(elapsedMilliseconds, 0) || elapsedMilliseconds < 0 || elapsedMilliseconds > 86400000 {
		return false
	}
	return p.recordObservation(stage, &elapsedMilliseconds)
}

func (p *qualificationProgress) recordObservation(stage string, frontendElapsed *float64) bool {
	if p == nil || !validQualificationStep(stage) {
		return false
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.finished || p.count >= 128 {
		return false
	}
	if frontendElapsed != nil && p.frontendSeen && *frontendElapsed < p.frontendLast {
		return false
	}
	elapsed := p.clock().Sub(p.origin)
	if elapsed < p.last {
		return false
	}
	p.last = elapsed
	p.count++
	value := map[string]any{"stage": stage}
	if frontendElapsed != nil {
		p.frontendSeen, p.frontendLast = true, *frontendElapsed
		value["frontend_elapsed_ms"] = *frontendElapsed
		value["frontend_origin"] = "first-reported-stage"
	}
	p.emit(value)
	return true
}

func (p *qualificationProgress) terminal(passed bool) {
	if p == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.finished {
		return
	}
	p.finished = true
	elapsed := p.clock().Sub(p.origin)
	if elapsed > p.last {
		p.last = elapsed
	}
	bound, watchdog, former := qualificationtiming.Seconds("gui_seconds"), qualificationtiming.Seconds("gui_watchdog_seconds"), 90
	if p.mode == "help" {
		bound, watchdog, former = qualificationtiming.Seconds("help_seconds"), qualificationtiming.Seconds("help_watchdog_seconds"), 35
	}
	classification := "failed"
	if passed {
		classification = "passed-within-former-bound"
		if p.last > time.Duration(former)*time.Second {
			classification = "passed-after-former-bound"
		}
	}
	p.emit(map[string]any{"stage": "terminal", "scope": "application-return", "passed": passed, "classification": classification, "effective_bound_seconds": watchdog, "capture_bound_seconds": bound, "former_bound_seconds": former})
}

// RecordQualificationTerminal is unbound and reports only the caller's actual
// post-cleanup outcome. Forced cleanup cannot turn a failed caller into a pass.
func RecordQualificationTerminal(b *Bridge, passed bool) {
	b.mu.RLock()
	progress := b.qualificationProgress
	b.mu.RUnlock()
	progress.terminal(passed)
}
