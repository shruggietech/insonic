// SPDX-License-Identifier: Apache-2.0
package voicemodels

import (
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/models"
	"strconv"
)

// Preparation evidence contains exact content hashes and current source references,
// never reusable audio, transcript text, or machine-local temporary paths.
func (s *Service) publishPreparation(ctx context.Context, claim catalog.Work, dataset catalog.SpeakerDataset, inputs []PreparedInput, summary Summary) (catalog.Publication, error) {
	refs := make([]PreparedInput, len(inputs))
	copy(refs, inputs)
	for i := range refs {
		refs[i].Path = ""
		refs[i].Text = ""
	}
	raw, err := json.Marshal(map[string]any{"contract_version": "1", "dataset_id": dataset.Dataset.ID, "dataset_manifest_digest": dataset.ManifestDigest, "recipe_digest": hash(dataset.Recipe), "inputs": refs, "summary": summary})
	if err != nil {
		return catalog.Publication{}, err
	}
	return s.publishBytes(ctx, models.StableID("speaker-preparation:"+claim.ID+":"+strconv.FormatInt(claim.Generation, 10)), raw, "speaker-preparation")
}
