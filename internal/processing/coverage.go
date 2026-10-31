// SPDX-License-Identifier: Apache-2.0
package processing

import (
	"github.com/shruggietech/insonic/internal/contracts"
	"math"
	"math/big"
	"sort"
	"strings"
)

const MaxCoverageIntervals = 1000000
const MaxCoverageRegions = 65536
const MaxCoverageObservations = 2 * MaxPlanWindows

type CoverageSpan struct {
	StartUS int64 `json:"start_us"`
	EndUS   int64 `json:"end_us"`
}
type CoverageObservation struct {
	WindowID            string         `json:"window_id"`
	OutputDigest        string         `json:"output_sha256"`
	TrackOrdinal        int            `json:"track_ordinal"`
	Core                CoverageSpan   `json:"core"`
	RecognitionStatus   string         `json:"recognition_status"`
	DiarizationStatus   string         `json:"diarization_status"`
	RecognitionSpeech   []CoverageSpan `json:"recognition_speech"`
	DiarizationSpeech   []CoverageSpan `json:"diarization_speech"`
	RecognitionNoSpeech bool           `json:"recognition_no_speech"`
	DiarizationNoSpeech bool           `json:"diarization_no_speech"`
}
type CoverageEvidence struct {
	Kind            string                `json:"kind"`
	Version         int                   `json:"evidence_version"`
	SourceDigest    string                `json:"source_sha256"`
	SourceRevision  int64                 `json:"source_revision"`
	SourceMapDigest string                `json:"source_map_sha256"`
	Observations    []CoverageObservation `json:"observations"`
}
type CoverageOptions struct {
	GapToleranceUS           int64   `json:"gap_tolerance_us,omitempty"`
	MinimumGapUS             int64   `json:"minimum_gap_us,omitempty"`
	BoundaryToleranceUS      int64   `json:"boundary_tolerance_us,omitempty"`
	DuplicateOverlapFraction float64 `json:"duplicate_overlap_fraction,omitempty"`
	After                    int     `json:"after,omitempty"`
	Limit                    int     `json:"limit,omitempty"`
}
type CoverageCue struct {
	ID, Text       string
	StartUS, EndUS int64
	TrackOrdinal   int // -1 is unresolved, never implicitly assigned to every lane.
	WindowIDs      []string
	VoiceIDs       []string
	Source         *CoverageSpan
}
type CoverageInput struct {
	RecordingID, SourceDigest, DocumentDigest, SourceMapDigest string
	SourceRevision, DocumentRevision                           int64
	Tracks                                                     []TrackPlan
	Cues                                                       []CoverageCue
	Evidence                                                   *CoverageEvidence
	Limitations                                                []string
}
type CoverageMetrics struct {
	SelectedAudioUS           int64 `json:"selected_audio_us"`
	CueCoverageUS             int64 `json:"cue_coverage_us"`
	ObservedSpeechUS          int64 `json:"observed_speech_us"`
	CoveredObservedSpeechUS   int64 `json:"covered_observed_speech_us"`
	UncoveredObservedSpeechUS int64 `json:"uncovered_observed_speech_us"`
	ObservedNoSpeechUS        int64 `json:"observed_no_speech_us"`
	UnknownActivityUS         int64 `json:"unknown_activity_us"`
	RepeatedTextOverlapUS     int64 `json:"repeated_text_overlap_us"`
	DuplicateOverlapUS        int64 `json:"duplicate_overlap_us"`
	BoundaryInconsistencies   int   `json:"boundary_inconsistencies"`
}
type CoverageFraction struct {
	NumeratorUS   int64    `json:"numerator_us"`
	DenominatorUS int64    `json:"denominator_us"`
	Denominator   string   `json:"denominator"`
	Fraction      *float64 `json:"fraction"`
}
type CoverageTrackReport struct {
	TrackOrdinal      int              `json:"track_ordinal"`
	Track             TrackPlan        `json:"track"`
	Selected          CoverageSpan     `json:"selected"`
	EvidenceAvailable bool             `json:"evidence_available"`
	Metrics           CoverageMetrics  `json:"metrics"`
	AudioCoverage     CoverageFraction `json:"audio_coverage"`
	SpeechCoverage    CoverageFraction `json:"speech_coverage"`
}
type CoverageRegion struct {
	TrackOrdinal int `json:"track_ordinal"`
	CoverageSpan
	Reason    string   `json:"reason"`
	CueIDs    []string `json:"cue_ids"`
	WindowIDs []string `json:"window_ids"`
	Uncertain bool     `json:"uncertain"`
}
type CoverageReport struct {
	Kind             string                `json:"kind"`
	Version          string                `json:"schema_version"`
	RecordingID      string                `json:"recording_id"`
	SourceDigest     string                `json:"source_sha256"`
	SourceRevision   int64                 `json:"source_revision"`
	SourceMapDigest  string                `json:"source_map_sha256"`
	DocumentDigest   string                `json:"document_sha256"`
	DocumentRevision int64                 `json:"document_revision"`
	EvidenceDigest   string                `json:"evidence_sha256"`
	OptionsDigest    string                `json:"options_sha256"`
	Options          CoverageOptions       `json:"options"`
	Tracks           []CoverageTrackReport `json:"tracks"`
	Regions          []CoverageRegion      `json:"regions"`
	TotalRegions     int                   `json:"total_regions"`
	Next             *int                  `json:"next"`
	Limitations      []string              `json:"limitations"`
}

func NormalizeCoverageOptions(o CoverageOptions) (CoverageOptions, error) {
	if o.MinimumGapUS == 0 {
		o.MinimumGapUS = 250000
	}
	if o.BoundaryToleranceUS == 0 {
		o.BoundaryToleranceUS = 1000
	}
	if o.DuplicateOverlapFraction == 0 {
		o.DuplicateOverlapFraction = .5
	}
	if o.Limit == 0 {
		o.Limit = 100
	}
	if o.GapToleranceUS < 0 || o.GapToleranceUS > 60000000 || o.MinimumGapUS < 1 || o.MinimumGapUS > 60000000 || o.BoundaryToleranceUS < 0 || o.BoundaryToleranceUS > 60000000 || math.IsNaN(o.DuplicateOverlapFraction) || o.DuplicateOverlapFraction <= 0 || o.DuplicateOverlapFraction > 1 || o.After < 0 || o.After > MaxCoverageRegions || o.Limit < 1 || o.Limit > 1000 {
		return o, contracts.Fail("invalid_request")
	}
	return o, nil
}

// TrackCoverageSpan retains the rational source origin until the microsecond boundary.
func TrackCoverageSpan(t TrackPlan) (CoverageSpan, error) {
	if t.StreamIndex < 0 || t.StreamIndex > 65535 || t.Channel != nil && (*t.Channel < 0 || *t.Channel > 63) || t.SourceSampleRate < 1 || t.SourceSampleRate > 768000 || t.SourceSampleCount < 0 || t.SourceSampleCount > 604800*t.SourceSampleRate || t.StartSample < 0 || t.EndSample < t.StartSample || t.EndSample > t.SourceSampleCount || len(t.StartNumerator) > 128 || len(t.StartDenominator) > 128 {
		return CoverageSpan{}, contracts.Fail("invalid_timing")
	}
	origin, ok := new(big.Rat).SetString(t.StartNumerator + "/" + t.StartDenominator)
	denominator, denOK := new(big.Int).SetString(t.StartDenominator, 10)
	if !ok || !denOK || denominator.Sign() <= 0 {
		return CoverageSpan{}, contracts.Fail("invalid_timing")
	}
	project := func(n int64, ceil bool) (int64, error) {
		r := new(big.Rat).Add(origin, new(big.Rat).SetFrac64(n, t.SourceSampleRate))
		r.Mul(r, big.NewRat(1000000, 1))
		return roundedRat(r, ceil)
	}
	a, e := project(t.StartSample, true)
	if e != nil {
		return CoverageSpan{}, e
	}
	b, e := project(t.EndSample, false)
	if t.StartSample == t.EndSample {
		b = a
	}
	if e != nil || b < a || a < math.MinInt64+604800000000 || b > math.MaxInt64-604800000000 {
		return CoverageSpan{}, contracts.Fail("invalid_timing")
	}
	return CoverageSpan{a, b}, nil
}
func WindowCoverageSpan(w WindowPlan) (CoverageSpan, error) {
	a, b, e := WindowCoreUS(w)
	if e != nil {
		return CoverageSpan{}, e
	}
	a, e = projectWindowUS(w.Mapping, a, true)
	if e != nil {
		return CoverageSpan{}, e
	}
	b, e = projectWindowUS(w.Mapping, b, false)
	return CoverageSpan{a, b}, e
}
func coverageClip(s, b CoverageSpan) (CoverageSpan, bool) {
	s.StartUS = max(s.StartUS, b.StartUS)
	s.EndUS = min(s.EndUS, b.EndUS)
	return s, s.EndUS > s.StartUS
}
func coverageUnion(spans []CoverageSpan) []CoverageSpan {
	out := append([]CoverageSpan(nil), spans...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].StartUS == out[j].StartUS {
			return out[i].EndUS < out[j].EndUS
		}
		return out[i].StartUS < out[j].StartUS
	})
	n := 0
	for _, s := range out {
		if s.EndUS <= s.StartUS {
			continue
		}
		if n > 0 && s.StartUS <= out[n-1].EndUS {
			out[n-1].EndUS = max(out[n-1].EndUS, s.EndUS)
		} else {
			out[n] = s
			n++
		}
	}
	return out[:n]
}
func coverageSubtract(a, b []CoverageSpan) []CoverageSpan {
	a, b = coverageUnion(a), coverageUnion(b)
	out := []CoverageSpan{}
	j := 0
	for _, s := range a {
		for j < len(b) && b[j].EndUS <= s.StartUS {
			j++
		}
		at := s.StartUS
		for k := j; k < len(b) && b[k].StartUS < s.EndUS; k++ {
			if b[k].StartUS > at {
				out = append(out, CoverageSpan{at, min(s.EndUS, b[k].StartUS)})
			}
			at = max(at, b[k].EndUS)
			if at >= s.EndUS {
				break
			}
		}
		if at < s.EndUS {
			out = append(out, CoverageSpan{at, s.EndUS})
		}
	}
	return out
}
func coverageDuration(spans []CoverageSpan) int64 {
	var n int64
	for _, s := range coverageUnion(spans) {
		n += s.EndUS - s.StartUS
	}
	return n
}
func coverageFraction(n, d int64, name string) CoverageFraction {
	v := CoverageFraction{NumeratorUS: n, DenominatorUS: d, Denominator: name}
	if d > 0 {
		x := float64(n) / float64(d)
		v.Fraction = &x
	}
	return v
}
func validCoverageStatus(s string) bool {
	return s == "executed" || s == "reused" || s == "unavailable"
}
func CoverageFromWindows(identity PlanIdentity, mapDigest string, windows []WindowEvidence, recognitionStatus, diarizationStatus string) (CoverageEvidence, error) {
	out := CoverageEvidence{Kind: "recording-coverage-evidence", Version: 1, SourceDigest: identity.SourceDigest, SourceRevision: identity.SourceRevision, SourceMapDigest: mapDigest, Observations: []CoverageObservation{}}
	if len(windows) > MaxPlanWindows || !validCoverageStatus(recognitionStatus) || !validCoverageStatus(diarizationStatus) {
		return out, contracts.Fail("invalid_request")
	}
	count := 0
	for _, v := range windows {
		if e := ValidateWindowEvidence(v); e != nil {
			return out, e
		}
		core, e := WindowCoverageSpan(v.Window)
		if e != nil {
			return out, e
		}
		o := CoverageObservation{WindowID: v.Window.ID, OutputDigest: v.OutputDigest, TrackOrdinal: v.Window.TrackOrdinal, Core: core, RecognitionStatus: recognitionStatus, DiarizationStatus: diarizationStatus, RecognitionSpeech: []CoverageSpan{}, DiarizationSpeech: []CoverageSpan{}}
		if recognitionStatus == "executed" {
			o.RecognitionNoSpeech = v.Recognition.NoSpeech
			for _, s := range v.Recognition.Segments {
				if strings.TrimSpace(s.Text) == "" {
					continue
				}
				a, e := projectWindowUS(v.Window.Mapping, s.StartUS, true)
				if e != nil {
					return out, e
				}
				b, e := projectWindowUS(v.Window.Mapping, s.EndUS, false)
				if e != nil {
					return out, e
				}
				if clipped, ok := coverageClip(CoverageSpan{a, b}, core); ok {
					o.RecognitionSpeech = append(o.RecognitionSpeech, clipped)
				}
			}
		}
		if diarizationStatus == "executed" {
			o.DiarizationNoSpeech = v.Diarization.NoSpeech
			for _, s := range v.Diarization.Turns {
				if clipped, ok := coverageClip(CoverageSpan{s.StartUS, s.EndUS}, core); ok {
					o.DiarizationSpeech = append(o.DiarizationSpeech, clipped)
				}
			}
		}
		count += len(o.RecognitionSpeech) + len(o.DiarizationSpeech)
		if count > MaxCoverageIntervals {
			return out, contracts.Fail("output_limit")
		}
		o.RecognitionSpeech = coverageUnion(o.RecognitionSpeech)
		o.DiarizationSpeech = coverageUnion(o.DiarizationSpeech)
		out.Observations = append(out.Observations, o)
	}
	return out, nil
}
func EvaluateCoverage(in CoverageInput, options CoverageOptions) (CoverageReport, error) {
	o, e := NormalizeCoverageOptions(options)
	r := CoverageReport{Kind: "recording-coverage", Version: contracts.Version, RecordingID: in.RecordingID, SourceDigest: in.SourceDigest, SourceRevision: in.SourceRevision, SourceMapDigest: in.SourceMapDigest, DocumentDigest: in.DocumentDigest, DocumentRevision: in.DocumentRevision, Options: o, Tracks: []CoverageTrackReport{}, Regions: []CoverageRegion{}, Limitations: append([]string{}, in.Limitations...)}
	if e != nil {
		return r, e
	}
	if len(in.Tracks) > 64 || len(in.Cues) > 65536 {
		return r, contracts.Fail("output_limit")
	}
	basis := o
	basis.After = 0
	basis.Limit = 0
	r.OptionsDigest, _ = canonicalPlanHash(basis)
	limitation := func(s string) {
		for _, v := range r.Limitations {
			if s == v {
				return
			}
		}
		r.Limitations = append(r.Limitations, s)
	}
	if len(in.Tracks) == 0 {
		limitation("source_track_intervals_unavailable")
	}
	present := in.Evidence != nil && in.Evidence.Kind == "recording-coverage-evidence" && in.Evidence.Version == 1 && in.Evidence.SourceDigest == in.SourceDigest && in.Evidence.SourceRevision == in.SourceRevision && in.Evidence.SourceMapDigest == in.SourceMapDigest
	if !present {
		limitation("acoustic_evidence_unavailable")
	} else {
		if len(in.Evidence.Observations) > MaxCoverageObservations {
			return r, contracts.Fail("output_limit")
		}
	}
	observations := make([][]CoverageObservation, len(in.Tracks))
	intervals := 0
	seen := map[string]bool{}
	observedByID := map[string]CoverageObservation{}
	if present {
		for _, v := range in.Evidence.Observations {
			if !contracts.ValidID(v.WindowID) || seen[v.WindowID] || !pinnedDigest.MatchString(v.OutputDigest) || v.TrackOrdinal < 0 || v.TrackOrdinal >= len(in.Tracks) || v.Core.EndUS < v.Core.StartUS || !validCoverageStatus(v.RecognitionStatus) || !validCoverageStatus(v.DiarizationStatus) {
				return r, contracts.Fail("invalid_request")
			}
			seen[v.WindowID] = true
			observedByID[v.WindowID] = v
			for _, stage := range []struct {
				status string
				speech []CoverageSpan
				none   bool
			}{{v.RecognitionStatus, v.RecognitionSpeech, v.RecognitionNoSpeech}, {v.DiarizationStatus, v.DiarizationSpeech, v.DiarizationNoSpeech}} {
				if stage.status != "executed" && (len(stage.speech) > 0 || stage.none) || stage.none && len(stage.speech) > 0 {
					return r, contracts.Fail("invalid_request")
				}
				intervals += len(stage.speech)
				if intervals > MaxCoverageIntervals {
					return r, contracts.Fail("output_limit")
				}
				for _, s := range stage.speech {
					if s.StartUS < v.Core.StartUS || s.EndUS > v.Core.EndUS || s.EndUS <= s.StartUS {
						return r, contracts.Fail("invalid_request")
					}
				}
			}
			if intervals > MaxCoverageIntervals {
				return r, contracts.Fail("output_limit")
			}
			observations[v.TrackOrdinal] = append(observations[v.TrackOrdinal], v)
		}
	}
	if present {
		r.EvidenceDigest, _ = canonicalPlanHash(in.Evidence)
	}
	cues := make([][]CoverageCue, len(in.Tracks))
	referenceCount, inputBytes := 0, 0
	corroborated := map[string]bool{}
	ids := map[string]bool{}
	for _, c := range in.Cues {
		if c.ID == "" || ids[c.ID] || c.EndUS < c.StartUS || c.StartUS < 0 || c.EndUS > math.MaxInt64-60000000 || len(c.WindowIDs) > MaxPlanWindows || len(c.VoiceIDs) > 1024 {
			return r, contracts.Fail("invalid_request")
		}
		referenceCount += len(c.WindowIDs) + len(c.VoiceIDs)
		inputBytes += len(c.ID) + len(c.Text)
		validRefs := len(c.WindowIDs) > 0
		for _, id := range c.WindowIDs {
			inputBytes += len(id)
			observation, ok := observedByID[id]
			_, overlaps := coverageClip(CoverageSpan{c.StartUS, c.EndUS}, observation.Core)
			if !ok || observation.TrackOrdinal != c.TrackOrdinal || !overlaps {
				validRefs = false
			}
		}
		for _, id := range c.VoiceIDs {
			inputBytes += len(id)
		}
		if referenceCount > MaxCoverageIntervals || inputBytes > 32<<20 {
			return r, contracts.Fail("output_limit")
		}
		corroborated[c.ID] = validRefs
		ids[c.ID] = true
		if c.TrackOrdinal < 0 {
			limitation("cue_source_track_unresolved")
			continue
		}
		if c.TrackOrdinal >= len(in.Tracks) {
			return r, contracts.Fail("invalid_request")
		}
		cues[c.TrackOrdinal] = append(cues[c.TrackOrdinal], c)
	}
	regions := []CoverageRegion{}
	outputRefs := 0
	add := func(lane int, s CoverageSpan, reason string, ids, windows []string, uncertain bool) error {
		outputRefs += len(ids) + len(windows)
		if outputRefs > MaxCoverageIntervals || len(regions) >= MaxCoverageRegions {
			return contracts.Fail("output_limit")
		}
		if s.EndUS < s.StartUS {
			return contracts.Fail("invalid_timing")
		}
		regions = append(regions, CoverageRegion{TrackOrdinal: lane, CoverageSpan: s, Reason: reason, CueIDs: append([]string{}, ids...), WindowIDs: append([]string{}, windows...), Uncertain: uncertain})
		return nil
	}
	comparisons := 0
	trackKeys := map[string]bool{}
	for lane, t := range in.Tracks {
		key := trackKey(t.StreamIndex, t.Channel)
		if trackKeys[key] {
			return r, contracts.Fail("invalid_request")
		}
		trackKeys[key] = true
		selected, err := TrackCoverageSpan(t)
		if err != nil {
			return r, err
		}
		tr := CoverageTrackReport{TrackOrdinal: lane, Track: t, Selected: selected, EvidenceAvailable: false}
		physicalTrack := t
		physicalTrack.StartSample = 0
		physicalTrack.EndSample = t.SourceSampleCount
		physical, err := TrackCoverageSpan(physicalTrack)
		if err != nil {
			return r, err
		}
		m := &tr.Metrics
		m.SelectedAudioUS = selected.EndUS - selected.StartUS
		speech, silence, text, expanded := []CoverageSpan{}, []CoverageSpan{}, []CoverageSpan{}, []CoverageSpan{}
		groups := map[string][]CoverageCue{}
		for _, v := range observations[lane] {
			if v.Core.StartUS < selected.StartUS || v.Core.EndUS > selected.EndUS {
				return r, contracts.Fail("invalid_request")
			}
			tr.EvidenceAvailable = tr.EvidenceAvailable || v.RecognitionStatus == "executed" || v.DiarizationStatus == "executed"
			speech = append(speech, v.RecognitionSpeech...)
			speech = append(speech, v.DiarizationSpeech...)
			if v.RecognitionNoSpeech || v.DiarizationNoSpeech {
				silence = append(silence, v.Core)
			}
			if v.RecognitionNoSpeech && len(v.DiarizationSpeech) > 0 || v.DiarizationNoSpeech && len(v.RecognitionSpeech) > 0 {
				limitation("recognition_diarization_disagreement")
			}
		}
		for _, c := range cues[lane] {
			if strings.TrimSpace(c.Text) == "" {
				continue
			}
			span := CoverageSpan{c.StartUS, c.EndUS}
			if clipped, ok := coverageClip(span, selected); ok {
				text = append(text, clipped)
			}
			if clipped, ok := coverageClip(CoverageSpan{c.StartUS - o.GapToleranceUS, c.EndUS + o.GapToleranceUS}, selected); ok {
				expanded = append(expanded, clipped)
			}
			if c.StartUS < selected.StartUS || c.EndUS > selected.EndUS {
				limitation("cue_outside_selected_audio")
				for _, gap := range coverageSubtract([]CoverageSpan{span}, []CoverageSpan{selected}) {
					if err = add(lane, gap, "outside_selected_scope", []string{c.ID}, c.WindowIDs, true); err != nil {
						return r, err
					}
				}
			}
			for _, gap := range coverageSubtract([]CoverageSpan{span}, []CoverageSpan{physical}) {
				if err = add(lane, gap, "missing_audio", []string{c.ID}, c.WindowIDs, true); err != nil {
					return r, err
				}
			}
			if c.Source != nil && (coverageAbsDifference(c.StartUS, c.Source.StartUS) > uint64(o.BoundaryToleranceUS) || coverageAbsDifference(c.EndUS, c.Source.EndUS) > uint64(o.BoundaryToleranceUS)) {
				m.BoundaryInconsistencies++
				if err = add(lane, span, "boundary_inconsistency", []string{c.ID}, c.WindowIDs, false); err != nil {
					return r, err
				}
			}
			key := coverageTextKey(c.Text)
			if key != "" {
				groups[key] = append(groups[key], c)
			}
		}
		speech = coverageUnion(speech)
		silence = coverageSubtract(silence, speech)
		gaps := coverageSubtract(speech, expanded)
		m.CueCoverageUS = coverageDuration(text)
		m.ObservedSpeechUS = coverageDuration(speech)
		m.UncoveredObservedSpeechUS = coverageDuration(gaps)
		m.CoveredObservedSpeechUS = m.ObservedSpeechUS - m.UncoveredObservedSpeechUS
		m.ObservedNoSpeechUS = coverageDuration(silence)
		m.UnknownActivityUS = coverageDuration(coverageSubtract([]CoverageSpan{selected}, append(append([]CoverageSpan{}, speech...), silence...)))
		sort.Slice(observations[lane], func(i, j int) bool { return observations[lane][i].Core.StartUS < observations[lane][j].Core.StartUS })
		observationStart := 0
		for _, gap := range gaps {
			if gap.EndUS-gap.StartUS < o.MinimumGapUS {
				continue
			}
			refs := []string{}
			for observationStart < len(observations[lane]) && observations[lane][observationStart].Core.EndUS <= gap.StartUS {
				observationStart++
			}
			for k := observationStart; k < len(observations[lane]) && observations[lane][k].Core.StartUS < gap.EndUS; k++ {
				v := observations[lane][k]
				comparisons++
				if comparisons > MaxCoverageIntervals {
					return r, contracts.Fail("output_limit")
				}
				if _, ok := coverageClip(gap, v.Core); ok {
					refs = append(refs, v.WindowID)
				}
			}
			if err = add(lane, gap, "uncovered_speech", nil, refs, false); err != nil {
				return r, err
			}
		}
		duplicate := []CoverageSpan{}
		repeated := []CoverageSpan{}
		for _, group := range groups {
			sort.Slice(group, func(i, j int) bool { return group[i].StartUS < group[j].StartUS })
			for i, a := range group {
				for j := i + 1; j < len(group) && group[j].StartUS < a.EndUS; j++ {
					comparisons++
					if comparisons > MaxCoverageIntervals {
						return r, contracts.Fail("output_limit")
					}
					b := group[j]
					overlap, ok := coverageClip(CoverageSpan{a.StartUS, a.EndUS}, CoverageSpan{b.StartUS, b.EndUS})
					if !ok || float64(overlap.EndUS-overlap.StartUS) < o.DuplicateOverlapFraction*float64(min(a.EndUS-a.StartUS, b.EndUS-b.StartUS)) {
						continue
					}
					comparisons += len(a.WindowIDs) + len(b.WindowIDs) + len(a.VoiceIDs) + len(b.VoiceIDs)
					if comparisons > MaxCoverageIntervals {
						return r, contracts.Fail("output_limit")
					}
					if clipped, ok := coverageClip(overlap, selected); ok {
						repeated = append(repeated, clipped)
					}
					windowSupport := corroborated[a.ID] && corroborated[b.ID] && coverageDistinctWindows(a.WindowIDs, b.WindowIDs)
					uncertain := !windowSupport || !coverageSameVoices(a.VoiceIDs, b.VoiceIDs)
					if windowSupport {
						if clipped, ok := coverageClip(overlap, selected); ok {
							duplicate = append(duplicate, clipped)
						}
					}
					if err = add(lane, overlap, "repeated_text", []string{a.ID, b.ID}, append(append([]string{}, a.WindowIDs...), b.WindowIDs...), uncertain); err != nil {
						return r, err
					}
				}
			}
		}
		m.DuplicateOverlapUS = coverageDuration(duplicate)
		m.RepeatedTextOverlapUS = coverageDuration(repeated)
		tr.AudioCoverage = coverageFraction(m.CueCoverageUS, m.SelectedAudioUS, "selected_audio_us")
		tr.SpeechCoverage = coverageFraction(m.CoveredObservedSpeechUS, m.ObservedSpeechUS, "observed_speech_us")
		r.Tracks = append(r.Tracks, tr)
	}
	sort.Slice(regions, func(i, j int) bool {
		a, b := regions[i], regions[j]
		if a.TrackOrdinal != b.TrackOrdinal {
			return a.TrackOrdinal < b.TrackOrdinal
		}
		if a.StartUS != b.StartUS {
			return a.StartUS < b.StartUS
		}
		if a.EndUS != b.EndUS {
			return a.EndUS < b.EndUS
		}
		if a.Reason != b.Reason {
			return a.Reason < b.Reason
		}
		return strings.Join(a.CueIDs, ",") < strings.Join(b.CueIDs, ",")
	})
	r.TotalRegions = len(regions)
	start := min(o.After, len(regions))
	end := min(start+o.Limit, len(regions))
	r.Regions = append(r.Regions, regions[start:end]...)
	if end < len(regions) {
		r.Next = &end
	}
	return r, nil
}
func coverageAbsDifference(a, b int64) uint64 {
	if a >= b {
		return uint64(a) - uint64(b)
	}
	return uint64(b) - uint64(a)
}
func coverageDistinctWindows(a, b []string) bool {
	if len(a) == 0 || len(b) == 0 {
		return false
	}
	seen := make(map[string]bool, len(a))
	for _, id := range a {
		seen[id] = true
	}
	for _, id := range b {
		if seen[id] {
			return false
		}
	}
	return true
}
func coverageTextKey(text string) string {
	words := []string{}
	for _, s := range strings.Fields(text) {
		if key := overlapToken(s); key != "" {
			words = append(words, key)
		}
	}
	return strings.Join(words, " ")
}
func coverageSameVoices(a, b []string) bool {
	// Two absent assignments provide no speaker evidence for automatic repair.
	if len(a) == 0 || len(a) != len(b) {
		return false
	}
	aa, bb := append([]string{}, a...), append([]string{}, b...)
	sort.Strings(aa)
	sort.Strings(bb)
	for i := range aa {
		if aa[i] != bb[i] {
			return false
		}
	}
	return true
}

// CoverageDigest preserves integer values and is independent of map iteration order.
func CoverageDigest(v any) string { d, _ := canonicalPlanHash(v); return d }

// RetainReusedCoverage retains the original verified observations for stages
// that did not execute again. No skipped stage is promoted into observed silence.
func RetainReusedCoverage(current CoverageEvidence, prior *CoverageEvidence, recognition, diarization bool) (CoverageEvidence, error) {
	if prior == nil || prior.SourceDigest != current.SourceDigest || prior.SourceRevision != current.SourceRevision || prior.SourceMapDigest != current.SourceMapDigest || prior.Version != 1 {
		return current, nil
	}
	out := current
	out.Observations = []CoverageObservation{}
	seen := map[string]bool{}
	for _, v := range current.Observations {
		if v.RecognitionStatus == "executed" || v.DiarizationStatus == "executed" {
			out.Observations = append(out.Observations, v)
			seen[v.WindowID] = true
		}
	}
	for _, v := range prior.Observations {
		if seen[v.WindowID] {
			continue
		}
		if !recognition {
			v.RecognitionStatus = "unavailable"
			v.RecognitionSpeech = []CoverageSpan{}
			v.RecognitionNoSpeech = false
		}
		if !diarization {
			v.DiarizationStatus = "unavailable"
			v.DiarizationSpeech = []CoverageSpan{}
			v.DiarizationNoSpeech = false
		}
		if v.RecognitionStatus == "executed" || v.DiarizationStatus == "executed" {
			out.Observations = append(out.Observations, v)
		}
	}
	if len(out.Observations) > MaxCoverageObservations {
		return out, contracts.Fail("output_limit")
	}
	return out, nil
}
