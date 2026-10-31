// SPDX-License-Identifier: Apache-2.0
package processing

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"

	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/library"
)

const MaxPlanWindows = 16384

type WindowOptions struct {
	CoreUS      int64            `json:"core_us,omitempty"`
	ContextUS   *int64           `json:"context_us,omitempty"`
	Concurrency int              `json:"concurrency,omitempty"`
	Tracks      []TrackSelection `json:"tracks,omitempty"`
}
type TrackSelection struct {
	StreamIndex int    `json:"stream_index"`
	Channel     *int   `json:"channel,omitempty"`
	StartSample int64  `json:"start_sample,omitempty"`
	EndSample   *int64 `json:"end_sample,omitempty"`
}
type PlanIdentity struct {
	SourceID            string `json:"source_id"`
	SourceDigest        string `json:"source_sha256"`
	SourceRevision      int64  `json:"source_revision"`
	DocumentDigest      string `json:"document_sha256,omitempty"`
	DocumentRevision    int64  `json:"document_revision,omitempty"`
	ConfigurationDigest string `json:"configuration_sha256"`
	ModelDigest         string `json:"model_sha256,omitempty"`
	AdapterID           string `json:"adapter_id,omitempty"`
	AdapterVersion      string `json:"adapter_version,omitempty"`
	DevicePolicy        string `json:"device_policy,omitempty"`
}
type TrackPlan struct {
	StreamIndex       int    `json:"stream_index"`
	Channel           *int   `json:"channel,omitempty"`
	SourceSampleRate  int64  `json:"source_sample_rate"`
	SourceSampleCount int64  `json:"source_sample_count"`
	StartNumerator    string `json:"start_numerator"`
	StartDenominator  string `json:"start_denominator"`
	StartSample       int64  `json:"start_sample"`
	EndSample         int64  `json:"end_sample"`
}
type WindowPlan struct {
	ID                 string    `json:"id"`
	Ordinal            int       `json:"ordinal"`
	TrackOrdinal       int       `json:"track_ordinal"`
	StreamIndex        int       `json:"stream_index"`
	Channel            *int      `json:"channel,omitempty"`
	SourceSampleRate   int64     `json:"source_sample_rate"`
	CoreStartSample    int64     `json:"core_start_sample"`
	CoreEndSample      int64     `json:"core_end_sample"`
	ContextStartSample int64     `json:"context_start_sample"`
	ContextEndSample   int64     `json:"context_end_sample"`
	MappedStartSample  int64     `json:"mapped_start_sample"`
	MappedEndSample    int64     `json:"mapped_end_sample"`
	Mapping            SourceMap `json:"mapping"`
}
type ProcessingPlan struct {
	Kind        string        `json:"kind"`
	Version     string        `json:"schema_version"`
	PlanVersion int           `json:"plan_version"`
	Identity    PlanIdentity  `json:"identity"`
	Options     WindowOptions `json:"options"`
	Tracks      []TrackPlan   `json:"tracks"`
	Windows     []WindowPlan  `json:"windows"`
	Digest      string        `json:"digest,omitempty"`
}

func NormalizeWindowOptions(o WindowOptions) (WindowOptions, error) {
	if o.CoreUS == 0 {
		o.CoreUS = 30_000_000
	}
	if o.Concurrency == 0 {
		o.Concurrency = 1
	}
	if o.CoreUS < 1_000_000 || o.CoreUS > 300_000_000 || o.Concurrency < 1 || o.Concurrency > 4 || len(o.Tracks) > 64 {
		return o, contracts.Fail("invalid_request")
	}
	context := min(int64(2_000_000), o.CoreUS/2)
	if o.ContextUS != nil {
		context = *o.ContextUS
	}
	if context < 0 || context > o.CoreUS/2 {
		return o, contracts.Fail("invalid_request")
	}
	o.ContextUS = &context
	o.Tracks = append([]TrackSelection(nil), o.Tracks...)
	seen := map[string]bool{}
	for i, s := range o.Tracks {
		if s.StreamIndex < 0 || s.StreamIndex > 65535 || s.StartSample < 0 || s.Channel != nil && (*s.Channel < 0 || *s.Channel > 63) || s.EndSample != nil && *s.EndSample <= s.StartSample {
			return o, contracts.Fail("invalid_request")
		}
		key := trackKey(s.StreamIndex, s.Channel)
		if seen[key] {
			return o, contracts.Fail("invalid_request")
		}
		seen[key] = true
		if s.Channel != nil {
			v := *s.Channel
			o.Tracks[i].Channel = &v
		}
		if s.EndSample != nil {
			v := *s.EndSample
			o.Tracks[i].EndSample = &v
		}
	}
	return o, nil
}
func trackKey(index int, channel *int) string {
	s := strconv.Itoa(index) + ":"
	if channel != nil {
		s += strconv.Itoa(*channel)
	}
	return s
}

func canonicalPlanHash(value any) (string, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return "", contracts.Fail("invalid_request")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var normalized any
	if decoder.Decode(&normalized) != nil {
		return "", contracts.Fail("invalid_request")
	}
	raw, err = json.Marshal(normalized)
	if err != nil {
		return "", contracts.Fail("invalid_request")
	}
	h := sha256.Sum256(raw)
	return hex.EncodeToString(h[:]), nil
}

// DigestWindowPlan uses sorted JSON object keys and exact integers. The digest
// field itself is absent from the hash input, allowing independent validation.
func DigestWindowPlan(plan ProcessingPlan) (string, error) {
	plan.Digest = ""
	return canonicalPlanHash(plan)
}

func BuildWindowPlan(identity PlanIdentity, options WindowOptions, tracks []TrackPlan) (ProcessingPlan, error) {
	plan := ProcessingPlan{Kind: "processing-window-plan", Version: contracts.Version, PlanVersion: 1, Identity: identity, Windows: []WindowPlan{}, Tracks: []TrackPlan{}}
	o, err := NormalizeWindowOptions(options)
	if err != nil {
		return plan, err
	}
	plan.Options = o
	if !contracts.ValidID(identity.SourceID) || !pinnedDigest.MatchString(identity.SourceDigest) || identity.SourceRevision < 1 || !pinnedDigest.MatchString(identity.ConfigurationDigest) || identity.DocumentDigest != "" && !pinnedDigest.MatchString(identity.DocumentDigest) || identity.DocumentRevision < 0 || identity.ModelDigest != "" && !pinnedDigest.MatchString(identity.ModelDigest) || len(tracks) == 0 || len(tracks) > 64 {
		return plan, contracts.Fail("invalid_request")
	}
	selected := tracks
	if len(o.Tracks) > 0 {
		selected = []TrackPlan{}
		for _, selection := range o.Tracks {
			found := false
			for _, t := range tracks {
				if t.StreamIndex == selection.StreamIndex && trackKey(t.StreamIndex, t.Channel) == trackKey(selection.StreamIndex, selection.Channel) {
					t.StartSample = selection.StartSample
					t.EndSample = t.SourceSampleCount
					if selection.EndSample != nil {
						t.EndSample = *selection.EndSample
					}
					selected = append(selected, t)
					found = true
					break
				}
			}
			if !found {
				return plan, contracts.Fail("unsupported_audio")
			}
		}
	}
	seen := map[string]bool{}
	for _, t := range selected {
		if t.EndSample == 0 && t.StartSample == 0 && len(o.Tracks) == 0 {
			t.EndSample = t.SourceSampleCount
		}
		origin, ok := new(big.Rat).SetString(t.StartNumerator + "/" + t.StartDenominator)
		den, denOK := new(big.Int).SetString(t.StartDenominator, 10)
		key := trackKey(t.StreamIndex, t.Channel)
		if !ok || !denOK || den.Sign() <= 0 || len(t.StartNumerator) > 128 || len(t.StartDenominator) > 128 || t.StreamIndex < 0 || t.StreamIndex > 65535 || t.Channel != nil && (*t.Channel < 0 || *t.Channel > 63) || t.SourceSampleRate < 1 || t.SourceSampleRate > 768000 || t.SourceSampleCount < 0 || t.SourceSampleCount > 604800*t.SourceSampleRate || t.StartSample < 0 || t.EndSample < t.StartSample || t.EndSample > t.SourceSampleCount || seen[key] {
			return plan, contracts.Fail("invalid_timing")
		}
		seen[key] = true
		t.StartNumerator = origin.Num().String()
		t.StartDenominator = origin.Denom().String()
		if t.Channel != nil {
			v := *t.Channel
			t.Channel = &v
		}
		plan.Tracks = append(plan.Tracks, t)
	}
	seed, err := canonicalPlanHash(struct {
		Identity PlanIdentity  `json:"identity"`
		Options  WindowOptions `json:"options"`
		Tracks   []TrackPlan   `json:"tracks"`
	}{identity, o, plan.Tracks})
	if err != nil {
		return plan, err
	}
	for trackOrdinal, t := range plan.Tracks {
		core := o.CoreUS * t.SourceSampleRate / 1_000_000
		overlap := *o.ContextUS * t.SourceSampleRate / 1_000_000
		if core < 1 {
			return plan, contracts.Fail("invalid_timing")
		}
		for start := t.StartSample; start < t.EndSample; {
			if len(plan.Windows) >= MaxPlanWindows {
				return plan, contracts.Fail("output_limit")
			}
			end := min(start+core, t.EndSample)
			left, right := max(t.StartSample, start-overlap), min(t.EndSample, end+overlap)
			mappedLeft := (left*16000 + t.SourceSampleRate - 1) / t.SourceSampleRate
			mappedRight := right * 16000 / t.SourceSampleRate
			if mappedRight < mappedLeft {
				mappedRight = mappedLeft
			}
			origin, _ := new(big.Rat).SetString(t.StartNumerator + "/" + t.StartDenominator)
			origin.Add(origin, new(big.Rat).SetFrac64(mappedLeft, 16000))
			mapping := SourceMap{StartNumerator: origin.Num().String(), StartDenominator: origin.Denom().String(), SampleCount: mappedRight - mappedLeft, SampleRate: 16000, StreamIndex: t.StreamIndex, Channel: t.Channel, Policy: "canonical-source-samples;inward-window;continuous-resample;source-start-rational"}
			ordinal := len(plan.Windows)
			plan.Windows = append(plan.Windows, WindowPlan{ID: library.DerivedID(seed, fmt.Sprintf("window:%d", ordinal)), Ordinal: ordinal, TrackOrdinal: trackOrdinal, StreamIndex: t.StreamIndex, Channel: t.Channel, SourceSampleRate: t.SourceSampleRate, CoreStartSample: start, CoreEndSample: end, ContextStartSample: left, ContextEndSample: right, MappedStartSample: mappedLeft, MappedEndSample: mappedRight, Mapping: mapping})
			start = end
		}
	}
	plan.Digest, err = DigestWindowPlan(plan)
	if raw, marshalErr := json.Marshal(plan); marshalErr != nil || len(raw) > 16<<20 {
		return plan, contracts.Fail("output_limit")
	}
	return plan, err
}

// WindowCoreUS reports the inward-projected core relative to the extracted
// 16 kHz window, retaining rational arithmetic until the microsecond boundary.
func WindowCoreUS(w WindowPlan) (int64, int64, error) {
	if w.SourceSampleRate < 1 || w.CoreStartSample < 0 || w.CoreEndSample < w.CoreStartSample || w.MappedStartSample < 0 || w.Mapping.SampleRate != 16000 {
		return 0, 0, contracts.Fail("invalid_timing")
	}
	origin := new(big.Rat).SetFrac64(w.MappedStartSample, 16000)
	project := func(sample int64, ceil bool) (int64, error) {
		r := new(big.Rat).SetFrac64(sample, w.SourceSampleRate)
		r.Sub(r, origin)
		r.Mul(r, big.NewRat(1000000, 1))
		return roundedRat(r, ceil)
	}
	a, e := project(w.CoreStartSample, true)
	if e != nil {
		return 0, 0, e
	}
	b, e := project(w.CoreEndSample, false)
	duration := w.Mapping.SampleCount * 1000000 / 16000
	a = min(duration, max(int64(0), a))
	b = min(duration, max(a, b))
	return a, b, e
}

// WindowSourceDurationNS bounds projected observations in the original source
// clock. Container duration can exclude a canonical track's preserved origin.
// Empty or wholly negative selections have no positive representable envelope.
func WindowSourceDurationNS(plan ProcessingPlan) (*int64, error) {
	maximum := new(big.Rat)
	for _, track := range plan.Tracks {
		if track.SourceSampleRate < 1 || track.SourceSampleRate > 768000 || track.StartSample < 0 || track.EndSample < track.StartSample || track.EndSample > track.SourceSampleCount || len(track.StartNumerator) > 128 || len(track.StartDenominator) > 128 {
			return nil, contracts.Fail("invalid_timing")
		}
		origin, ok := new(big.Rat).SetString(track.StartNumerator + "/" + track.StartDenominator)
		denominator, denominatorOK := new(big.Int).SetString(track.StartDenominator, 10)
		if !ok || !denominatorOK || denominator.Sign() <= 0 {
			return nil, contracts.Fail("invalid_timing")
		}
		if track.StartSample == track.EndSample {
			continue
		}
		end := origin.Add(origin, new(big.Rat).SetFrac64(track.EndSample, track.SourceSampleRate))
		if end.Cmp(maximum) > 0 {
			maximum.Set(end)
		}
	}
	if maximum.Sign() <= 0 {
		return nil, nil
	}
	maximum.Mul(maximum, big.NewRat(1_000_000_000, 1))
	duration, err := roundedRat(maximum, true)
	if err != nil {
		return nil, err
	}
	return &duration, nil
}
