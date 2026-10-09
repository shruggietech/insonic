// SPDX-License-Identifier: Apache-2.0
package voicemodels

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/models"
	"github.com/shruggietech/insonic/internal/processing"
	"github.com/shruggietech/insonic/internal/speakers"
	"github.com/shruggietech/insonic/internal/workspace"
)

func validateRecipe(r Recipe) error {
	if r.RecordingID != "" && !contracts.ValidID(r.RecordingID) || len(r.RecordingIDs) > 1000 || len(r.Languages) > 64 || len(r.ExcludedReferences) > 10000 || r.Channel != nil && (*r.Channel < 0 || *r.Channel > 63) || r.MinDurationUS < 0 || r.MaxDurationUS < 0 || r.MaxDurationUS > 0 && r.MaxDurationUS < r.MinDurationUS {
		return contracts.Fail("invalid_request")
	}
	for _, v := range r.RecordingIDs {
		if !contracts.ValidID(v) {
			return contracts.Fail("invalid_request")
		}
	}
	for _, v := range r.Languages {
		if !text(v, 80) {
			return contracts.Fail("invalid_request")
		}
	}
	for _, v := range r.ExcludedReferences {
		if !text(v, 1024) {
			return contracts.Fail("invalid_request")
		}
	}
	for _, v := range []*float64{r.MinRMS, r.MaxClippingFraction} {
		if v != nil && (math.IsNaN(*v) || math.IsInf(*v, 0) || *v < 0 || *v > 1) {
			return contracts.Fail("invalid_request")
		}
	}
	return nil
}
func referenceKey(ref catalog.CurrentReference) string {
	return ref.RecordingID + "/" + ref.CueID + "/" + ref.LocalSpeakerID
}
func contains(v []string, needle string) bool {
	for _, item := range v {
		if item == needle {
			return true
		}
	}
	return false
}
func (s *Service) resolve(ctx context.Context, refs []catalog.CurrentReference, epoch string, recipe Recipe, speakerID string) ([]speakers.SelectedEvidence, Summary, error) {
	out := []speakers.SelectedEvidence{}
	summary := Summary{References: len(refs), Diagnostics: []processing.Diagnostic{}}
	counts := map[string]int64{}
	filter := catalog.SpeakerSelection{SpeakerID: speakerID, ConfirmedOnly: true}
	for start := 0; start < len(refs); start += 100 {
		end := start + 100
		if end > len(refs) {
			end = len(refs)
		}
		batch := []catalog.ResolvedEvidence{}
		for _, ref := range refs[start:end] {
			resolved, err := s.Catalog.ResolveEvidence(ctx, ref)
			if err != nil {
				return nil, summary, err
			}
			if contains(recipe.ExcludedReferences, referenceKey(ref)) {
				counts["explicit_reference_excluded"]++
				continue
			}
			if recipe.RecordingID != "" && recipe.RecordingID != ref.RecordingID || len(recipe.RecordingIDs) > 0 && !contains(recipe.RecordingIDs, ref.RecordingID) {
				counts["recording_filter_excluded"]++
				continue
			}
			if len(recipe.Languages) > 0 {
				recording, err := s.Catalog.Recording(ctx, ref.RecordingID)
				if err != nil {
					return nil, summary, err
				}
				var doc struct {
					Metadata struct {
						Language *string `json:"language"`
					} `json:"metadata"`
				}
				if json.Unmarshal(recording.Document, &doc) != nil {
					return nil, summary, contracts.Fail("conflict")
				}
				if doc.Metadata.Language == nil || !contains(recipe.Languages, *doc.Metadata.Language) {
					counts["language_filter_excluded"]++
					continue
				}
			}
			batch = append(batch, resolved)
		}
		selected, err := speakers.SelectWithPrior(batch, func(ref catalog.CurrentReference) ([]catalog.PriorEvidenceComparison, error) {
			return s.Catalog.ComparePriorSpeakerEvidence(ctx, filter, ref, epoch)
		})
		if err != nil {
			return nil, summary, err
		}
		for _, d := range selected.Diagnostics {
			counts[d.Code] += int64(d.Count)
		}
		for _, item := range selected.Items {
			kept := item
			kept.Intervals = nil
			comparisons, err := s.Catalog.ComparePriorSpeakerEvidence(ctx, filter, item.Reference, epoch)
			if err != nil {
				return nil, summary, err
			}
			overlap := false
			for _, c := range comparisons {
				overlap = overlap || c.Overlap
			}
			if recipe.ExcludeOverlap && overlap {
				counts["overlap_filter_excluded"]++
				continue
			}
			for _, span := range item.Intervals {
				first, ok := new(big.Rat).SetString(span.Start)
				if !ok {
					return nil, summary, contracts.Fail("invalid_timing")
				}
				last, ok := new(big.Rat).SetString(span.End)
				if !ok {
					return nil, summary, contracts.Fail("invalid_timing")
				}
				duration := new(big.Rat).Sub(last, first)
				duration.Mul(duration, big.NewRat(1000000, 1))
				us := new(big.Int).Quo(duration.Num(), duration.Denom())
				if !us.IsInt64() {
					return nil, summary, contracts.Fail("input_limit")
				}
				if us.Int64() < recipe.MinDurationUS || recipe.MaxDurationUS > 0 && us.Int64() > recipe.MaxDurationUS {
					counts["duration_filter_excluded"]++
					continue
				}
				kept.Intervals = append(kept.Intervals, span)
				summary.DurationUS += us.Int64()
			}
			if len(kept.Intervals) > 0 {
				out = append(out, kept)
				summary.Included++
			}
		}
	}
	summary.Excluded = summary.References - summary.Included
	keys := []string{}
	for code := range counts {
		keys = append(keys, code)
	}
	sort.Strings(keys)
	for _, code := range keys {
		summary.Diagnostics = append(summary.Diagnostics, processing.Diagnostic{Code: code, Count: counts[code]})
	}
	return out, summary, nil
}
func (s *Service) CreateDataset(ctx context.Context, op string, options DatasetOptions) (catalog.SpeakerDataset, error) {
	var empty catalog.SpeakerDataset
	if !contracts.ValidID(op) || !contracts.ValidID(options.SpeakerID) || validateRecipe(options.Recipe) != nil {
		return empty, contracts.Fail("invalid_request")
	}
	// Replay reads the original election, never replaces it with current evidence.
	if prior, err := s.Catalog.SpeakerDataset(ctx, catalog.SpeakerDatasetID(op, options.SpeakerID)); err == nil {
		wanted, _ := json.Marshal(options.Recipe)
		if string(wanted) != string(prior.Recipe) {
			return empty, contracts.Fail("conflict")
		}
		return prior, nil
	}
	selection := catalog.SpeakerSelection{SpeakerID: options.SpeakerID, ConfirmedOnly: true, Limit: 100}
	refs := []catalog.CurrentReference{}
	epoch := ""
	var revision int64
	for {
		page, err := s.Catalog.CurrentSpeakerReferences(ctx, selection)
		if err != nil {
			return empty, err
		}
		if epoch != "" && page.Epoch != epoch {
			return empty, contracts.Fail("conflict")
		}
		epoch = page.Epoch
		revision = page.Revision
		refs = append(refs, page.References...)
		if len(refs) > 100000 {
			return empty, contracts.Fail("input_limit")
		}
		if page.Next == "" {
			break
		}
		selection.Cursor = page.Next
	}
	selected, summary, err := s.resolve(ctx, refs, epoch, options.Recipe, options.SpeakerID)
	if err != nil {
		return empty, err
	}
	included := make([]catalog.CurrentReference, 0, len(selected))
	for _, item := range selected {
		included = append(included, item.Reference)
	}
	speaker, err := s.Catalog.Speaker(ctx, options.SpeakerID)
	if err != nil {
		return empty, err
	}
	id := catalog.SpeakerDatasetID(op, options.SpeakerID)
	manifest, err := BuildDatasetManifest(s.Artifacts.Workspace.Config.WorkspaceID, id, speaker.Speaker, revision, options.Recipe, included, summary)
	if err != nil {
		return empty, err
	}
	publication, err := s.publishBytes(ctx, models.StableID("speaker-dataset-manifest:"+op), manifest, "speaker-dataset")
	if err != nil {
		return empty, err
	}
	recipeRaw, _ := json.Marshal(options.Recipe)
	summaryRaw, _ := json.Marshal(summary)
	got, err := s.Catalog.CreateSpeakerDataset(ctx, op, options.SpeakerID, included, epoch, recipeRaw, summaryRaw, publication.ID)
	if err != nil {
		s.retire([]string{publication.ID}, op)
	}
	return got, err
}
func sampleBounds(start, end string, mapping processing.SourceMap) (int64, int64, error) {
	first, ok := new(big.Rat).SetString(start)
	if !ok {
		return 0, 0, contracts.Fail("invalid_timing")
	}
	last, ok := new(big.Rat).SetString(end)
	if !ok {
		return 0, 0, contracts.Fail("invalid_timing")
	}
	origin, ok := new(big.Rat).SetString(mapping.StartNumerator + "/" + mapping.StartDenominator)
	if !ok || mapping.SampleRate < 1 || mapping.SampleCount < 1 {
		return 0, 0, contracts.Fail("invalid_timing")
	}
	first.Sub(first, origin)
	last.Sub(last, origin)
	first.Mul(first, new(big.Rat).SetInt64(mapping.SampleRate))
	last.Mul(last, new(big.Rat).SetInt64(mapping.SampleRate))
	a := new(big.Int)
	rem := new(big.Int)
	a.QuoRem(first.Num(), first.Denom(), rem)
	if rem.Sign() > 0 {
		a.Add(a, big.NewInt(1))
	}
	b := new(big.Int).Quo(last.Num(), last.Denom())
	if first.Sign() < 0 || last.Sign() < 0 || !a.IsInt64() || !b.IsInt64() || a.Int64() >= b.Int64() || b.Int64() > mapping.SampleCount {
		return 0, 0, contracts.Fail("invalid_timing")
	}
	return a.Int64(), b.Int64(), nil
}
func wavData(f *os.File) (int64, int64, error) {
	header := make([]byte, 12)
	if _, err := io.ReadFull(f, header); err != nil || string(header[:4]) != "RIFF" || string(header[8:]) != "WAVE" {
		return 0, 0, contracts.Fail("invalid_audio")
	}
	format := false
	for i := 0; i < 64; i++ {
		chunk := make([]byte, 8)
		if _, err := io.ReadFull(f, chunk); err != nil {
			return 0, 0, contracts.Fail("invalid_audio")
		}
		n := int64(binary.LittleEndian.Uint32(chunk[4:]))
		position, _ := f.Seek(0, io.SeekCurrent)
		if string(chunk[:4]) == "fmt " {
			if n < 16 || n > 256 {
				return 0, 0, contracts.Fail("invalid_audio")
			}
			raw := make([]byte, n)
			if _, err := io.ReadFull(f, raw); err != nil {
				return 0, 0, contracts.Fail("invalid_audio")
			}
			format = binary.LittleEndian.Uint16(raw) == 1 && binary.LittleEndian.Uint16(raw[2:]) == 1 && binary.LittleEndian.Uint32(raw[4:]) == 16000 && binary.LittleEndian.Uint16(raw[14:]) == 16
			if n%2 != 0 {
				f.Seek(1, io.SeekCurrent)
			}
		} else if string(chunk[:4]) == "data" {
			if !format || n%2 != 0 {
				return 0, 0, contracts.Fail("invalid_audio")
			}
			return position, n, nil
		} else {
			if _, err := f.Seek(n+n%2, io.SeekCurrent); err != nil {
				return 0, 0, contracts.Fail("invalid_audio")
			}
		}
	}
	return 0, 0, contracts.Fail("invalid_audio")
}
func writeWAV(path string, pcm []byte) error {
	if len(pcm) > math.MaxUint32-36 || len(pcm)%2 != 0 {
		return contracts.Fail("input_limit")
	}
	header := make([]byte, 44)
	copy(header, "RIFF")
	binary.LittleEndian.PutUint32(header[4:], uint32(len(pcm)+36))
	copy(header[8:], "WAVEfmt ")
	binary.LittleEndian.PutUint32(header[16:], 16)
	binary.LittleEndian.PutUint16(header[20:], 1)
	binary.LittleEndian.PutUint16(header[22:], 1)
	binary.LittleEndian.PutUint32(header[24:], 16000)
	binary.LittleEndian.PutUint32(header[28:], 32000)
	binary.LittleEndian.PutUint16(header[32:], 2)
	binary.LittleEndian.PutUint16(header[34:], 16)
	copy(header[36:], "data")
	binary.LittleEndian.PutUint32(header[40:], uint32(len(pcm)))
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return contracts.Fail("unavailable")
	}
	defer f.Close()
	if _, err = f.Write(header); err == nil {
		_, err = f.Write(pcm)
	}
	if err == nil {
		err = f.Sync()
	}
	return err
}
func (s *Service) prepare(ctx context.Context, dataset catalog.SpeakerDataset, options TrainOptions, directory string) ([]PreparedInput, Summary, error) {
	var recipe Recipe
	if strict(dataset.Recipe, &recipe) != nil {
		return nil, Summary{}, contracts.Fail("conflict")
	}
	selected, summary, err := s.resolve(ctx, dataset.References, dataset.Epoch, recipe, dataset.Dataset.SpeakerID)
	if err != nil {
		return nil, summary, err
	}
	inputs := []PreparedInput{}
	limits, _ := normalizeLimits(options.Adapter.Limits)
	total := int64(0)
	audioCache := map[string]*Audio{}
	unusable := map[string]bool{}
	preparedReferences := map[string]bool{}
	summary.DurationUS = 0
	defer func() {
		for _, audio := range audioCache {
			if audio.Close != nil {
				audio.Close()
			}
		}
	}()
	for _, item := range selected {
		if ctx.Err() != nil {
			return nil, summary, contracts.Fail("cancelled")
		}
		if unusable[item.Reference.RecordingID] {
			continue
		}
		audio := audioCache[item.Reference.RecordingID]
		if audio == nil {
			entry, err := s.Catalog.Library(ctx, item.Reference.RecordingID)
			if err != nil {
				return nil, summary, err
			}
			if entry.Digest != item.Reference.SourceDigest {
				return nil, summary, contracts.Fail("conflict")
			}
			if s.Prepare == nil {
				return nil, summary, contracts.Fail("unavailable")
			}
			recording, err := s.Catalog.Recording(ctx, item.Reference.RecordingID)
			if err != nil {
				return nil, summary, err
			}
			var source processing.SourceMap
			if json.Unmarshal(recording.SourceMap, &source) != nil {
				return nil, summary, contracts.Fail("conflict")
			}
			channel := source.Channel
			if recipe.Channel != nil {
				channel = recipe.Channel
			}
			audio, err = s.Prepare(ctx, entry, processing.AudioOptions{StreamIndex: &source.StreamIndex, Channel: channel})
			if err != nil {
				var failure *contracts.Error
				if errors.As(err, &failure) && (failure.Code == "invalid_audio" || failure.Code == "unsupported_audio" || failure.Code == "timing_unavailable") {
					unusable[item.Reference.RecordingID] = true
					summary.Diagnostics = append(summary.Diagnostics, processing.Diagnostic{Code: "unusable_source_excluded", Count: 1})
					continue
				}
				return nil, summary, err
			}
			audioCache[item.Reference.RecordingID] = audio
		}
		for _, span := range item.Intervals {
			first, last, err := sampleBounds(span.Start, span.End, audio.SourceMap)
			if err != nil {
				return nil, summary, err
			}
			size := (last - first) * 2
			if size > 256<<20 || total+size+44 > limits.MaxInputBytes {
				return nil, summary, contracts.Fail("input_limit")
			}
			source, err := os.Open(audio.Path)
			if err != nil {
				return nil, summary, contracts.Fail("unavailable")
			}
			offset, dataSize, err := wavData(source)
			if err != nil || last*2 > dataSize {
				source.Close()
				summary.Diagnostics = append(summary.Diagnostics, processing.Diagnostic{Code: "unusable_clip_excluded", Count: 1})
				continue
			}
			pcm := make([]byte, size)
			_, err = io.ReadFull(io.NewSectionReader(source, offset+first*2, size), pcm)
			source.Close()
			if err != nil {
				summary.Diagnostics = append(summary.Diagnostics, processing.Diagnostic{Code: "unusable_clip_excluded", Count: 1})
				continue
			}
			sum := 0.0
			clipped := 0
			for i := 0; i < len(pcm); i += 2 {
				value := int16(binary.LittleEndian.Uint16(pcm[i:]))
				scaled := float64(value) / 32768
				sum += scaled * scaled
				if value == 32767 || value == -32768 {
					clipped++
				}
			}
			rms := math.Sqrt(sum / float64(last-first))
			fraction := float64(clipped) / float64(last-first)
			if rms == 0 || recipe.MinRMS != nil && rms < *recipe.MinRMS || recipe.MaxClippingFraction != nil && fraction > *recipe.MaxClippingFraction {
				summary.Diagnostics = append(summary.Diagnostics, processing.Diagnostic{Code: "signal_filter_excluded", Count: 1})
				continue
			}
			id := fmt.Sprintf("clip-%06d", len(inputs))
			path := filepath.Join(directory, id+".wav")
			if err = writeWAV(path, pcm); err != nil {
				return nil, summary, err
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				return nil, summary, contracts.Fail("unavailable")
			}
			mapping := audio.SourceMap
			mapping.SampleCount = last - first
			mapping.StartNumerator = "0"
			mapping.StartDenominator = "1"
			mapping.Policy = "selected-current-interval;source-clock-projection"
			input := PreparedInput{ID: id, Path: path, SHA256: hash(raw), Size: int64(len(raw)), DurationUS: (last - first) * 1000000 / audio.SourceMap.SampleRate, SourceMap: mapping}
			ref := item.Reference
			input.SourceReference = &ref
			origin, _ := new(big.Rat).SetString(audio.SourceMap.StartNumerator + "/" + audio.SourceMap.StartDenominator)
			origin.Add(origin, new(big.Rat).SetFrac(big.NewInt(first), big.NewInt(audio.SourceMap.SampleRate)))
			input.OriginalStartNumerator = origin.Num().String()
			input.OriginalStartDenominator = origin.Denom().String()
			if options.Adapter.NeedsText {
				resolved, err := s.Catalog.ResolveEvidence(ctx, item.Reference)
				if err != nil {
					return nil, summary, err
				}
				var cue struct {
					Payload struct {
						PlainText string `json:"plain_text"`
					} `json:"payload"`
				}
				if json.Unmarshal(resolved.Cue, &cue) != nil || strings.TrimSpace(cue.Payload.PlainText) == "" {
					return nil, summary, contracts.Fail("text_unavailable")
				}
				input.Text = cue.Payload.PlainText
			}
			inputs = append(inputs, input)
			total += input.Size
			summary.DurationUS += input.DurationUS
			preparedReferences[referenceKey(item.Reference)] = true
		}
	}
	summary.Included = len(preparedReferences)
	summary.Excluded = summary.References - summary.Included
	if len(inputs) == 0 {
		return nil, summary, contracts.Fail("empty_dataset")
	}
	return inputs, summary, nil
}
func (s *Service) scratch() (string, error) {
	dir, err := os.MkdirTemp(s.Artifacts.Workspace.Control, "speaker-work-")
	if err != nil {
		return "", contracts.Fail("unavailable")
	}
	if err = workspace.SecureDirectory(dir, true); err != nil {
		os.RemoveAll(dir)
		return "", err
	}
	return dir, nil
}
