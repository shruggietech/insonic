// SPDX-License-Identifier: Apache-2.0
package processing

import (
	"reflect"
	"strings"
	"testing"
)

func windowTestIdentity() PlanIdentity {
	return PlanIdentity{SourceID: "11111111-1111-4111-8111-111111111111", SourceDigest: strings.Repeat("a", 64), SourceRevision: 1, ConfigurationDigest: strings.Repeat("b", 64)}
}
func windowTestTrack(count int64) TrackPlan {
	return TrackPlan{StreamIndex: 0, SourceSampleRate: 48000, SourceSampleCount: count, StartNumerator: "96001", StartDenominator: "48000", EndSample: count}
}

func TestWindowOptionsDefaultsAndExplicitZero(t *testing.T) {
	o, err := NormalizeWindowOptions(WindowOptions{})
	if err != nil || o.CoreUS != 30_000_000 || o.ContextUS == nil || *o.ContextUS != 2_000_000 || o.Concurrency != 1 {
		t.Fatal(o, err)
	}
	zero := int64(0)
	o, err = NormalizeWindowOptions(WindowOptions{CoreUS: 1_000_000, ContextUS: &zero})
	if err != nil || *o.ContextUS != 0 {
		t.Fatal(o, err)
	}
	o, err = NormalizeWindowOptions(WindowOptions{CoreUS: 1_000_000})
	if err != nil || *o.ContextUS != 500_000 {
		t.Fatal(o, err)
	}
	for _, o := range []WindowOptions{{CoreUS: 999999}, {CoreUS: 300000001}, {Concurrency: 5}, {Concurrency: -1}, {CoreUS: 1000000, ContextUS: func() *int64 { v := int64(500001); return &v }()}} {
		if _, err := NormalizeWindowOptions(o); err == nil {
			t.Fatal("invalid options", o)
		}
	}
}

func TestWindowPlanExactCoverageAndIdentity(t *testing.T) {
	for _, count := range []int64{0, 1, 48000, 30 * 48000, 31 * 48000, 61*48000 + 1} {
		track := windowTestTrack(count)
		p, err := BuildWindowPlan(windowTestIdentity(), WindowOptions{}, []TrackPlan{track})
		if err != nil {
			t.Fatal(err)
		}
		position := int64(0)
		for i, w := range p.Windows {
			if w.Ordinal != i || w.CoreStartSample != position || w.CoreEndSample <= position || w.ContextStartSample > w.CoreStartSample || w.ContextEndSample < w.CoreEndSample || w.ContextEndSample > count {
				t.Fatal(w)
			}
			position = w.CoreEndSample
		}
		if position != count {
			t.Fatalf("coverage %d != %d", position, count)
		}
		again, err := BuildWindowPlan(windowTestIdentity(), WindowOptions{}, []TrackPlan{track})
		if err != nil || !reflect.DeepEqual(p, again) {
			t.Fatal("nondeterministic plan")
		}
		changed := windowTestIdentity()
		changed.ConfigurationDigest = strings.Repeat("c", 64)
		other, err := BuildWindowPlan(changed, WindowOptions{}, []TrackPlan{track})
		if err != nil || p.Digest == other.Digest {
			t.Fatal("config did not invalidate plan")
		}
	}
}

func TestWindowPlanUnequalTracksAndSourceSelection(t *testing.T) {
	t1, t2 := windowTestTrack(31*48000), windowTestTrack(48000)
	t2.StreamIndex = 2
	t2.StartNumerator = "-1"
	t2.StartDenominator = "3"
	end := int64(48001)
	p, err := BuildWindowPlan(windowTestIdentity(), WindowOptions{Tracks: []TrackSelection{{StreamIndex: 0, StartSample: 1, EndSample: &end}, {StreamIndex: 2}}}, []TrackPlan{t1, t2})
	if err != nil || len(p.Windows) != 2 || p.Windows[0].CoreStartSample != 1 || p.Windows[0].CoreEndSample != 48001 || p.Windows[1].TrackOrdinal != 1 {
		t.Fatal(p, err)
	}
	if p.Windows[0].Mapping.StartNumerator == "" || p.Windows[0].SourceSampleRate != 48000 {
		t.Fatal("mapping missing")
	}
	for _, tracks := range [][]TrackPlan{{{SourceSampleRate: 0}}, {{SourceSampleRate: 48000, SourceSampleCount: -1}}, {{SourceSampleRate: 48000, SourceSampleCount: 1, StartNumerator: "1", StartDenominator: "0"}}} {
		if _, err := BuildWindowPlan(windowTestIdentity(), WindowOptions{}, tracks); err == nil {
			t.Fatal("invalid track accepted")
		}
	}
}

func TestWindowCoreProjection(t *testing.T) {
	p, err := BuildWindowPlan(windowTestIdentity(), WindowOptions{}, []TrackPlan{windowTestTrack(61 * 48000)})
	if err != nil {
		t.Fatal(err)
	}
	a, b, err := WindowCoreUS(p.Windows[1])
	if err != nil || a != 2_000_000 || b != 32_000_000 {
		t.Fatal(a, b, err)
	}
	end := int64(2)
	p, err = BuildWindowPlan(windowTestIdentity(), WindowOptions{Tracks: []TrackSelection{{StreamIndex: 0, StartSample: 1, EndSample: &end}}}, []TrackPlan{windowTestTrack(10)})
	if err != nil {
		t.Fatal(err)
	}
	a, b, err = WindowCoreUS(p.Windows[0])
	if err != nil || a != 0 || b != 0 {
		t.Fatal("subsample core must be empty", a, b, err)
	}
}

func TestWindowSourceDurationUsesOriginalClockEnvelope(t *testing.T) {
	for _, tc := range []struct {
		name, numerator, denominator string
		rate, end                    int64
		want                         *int64
	}{
		{"positive", "600", "1", 48000, 96000, func() *int64 { v := int64(602000000000); return &v }()},
		{"negative-to-positive", "-1", "1", 48000, 96000, func() *int64 { v := int64(1000000000); return &v }()},
		{"negative", "-3", "1", 48000, 96000, nil},
		{"fractional", "1", "7", 3, 1, func() *int64 { v := int64(476190477); return &v }()},
		{"empty", "600", "1", 48000, 0, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			plan := ProcessingPlan{Tracks: []TrackPlan{{StartNumerator: tc.numerator, StartDenominator: tc.denominator, SourceSampleRate: tc.rate, SourceSampleCount: tc.end, EndSample: tc.end}}}
			got, err := WindowSourceDurationNS(plan)
			if err != nil || !reflect.DeepEqual(got, tc.want) {
				t.Fatal(got, tc.want, err)
			}
		})
	}
	track := windowTestTrack(48000)
	track.StartNumerator = "99999999999999999999999999999999999999999999999"
	if _, err := WindowSourceDurationNS(ProcessingPlan{Tracks: []TrackPlan{track}}); err == nil {
		t.Fatal("overflowing source clock accepted")
	}
	track.StartDenominator = "0"
	if _, err := WindowSourceDurationNS(ProcessingPlan{Tracks: []TrackPlan{track}}); err == nil {
		t.Fatal("invalid source denominator accepted")
	}
	first, second := windowTestTrack(48000), windowTestTrack(96000)
	first.StartNumerator, first.StartDenominator = "10", "1"
	second.StartNumerator, second.StartDenominator = "0", "1"
	got, err := WindowSourceDurationNS(ProcessingPlan{Tracks: []TrackPlan{first, second}})
	if err != nil || got == nil || *got != 11_000_000_000 {
		t.Fatal("not maximum source endpoint", got, err)
	}
}
