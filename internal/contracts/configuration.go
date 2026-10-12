// SPDX-License-Identifier: Apache-2.0
package contracts

// ConfigurationOperation identifies shared pipeline, speaker and term operations.
func ConfigurationOperation(operation string) bool {
	switch operation {
	case "import-presets.create","import-presets.list","import-presets.show","import-presets.update","import-presets.delete",
 "pipelines.list", "pipelines.show", "pipelines.set", "pipelines.inspect",
		"speakers.list", "speakers.show", "speakers.set", "speakers.aliases", "speakers.select", "speakers.diagnostics",
		"terms.list", "terms.show", "terms.set", "terms.compile":
		return true
	}
	return false
}

// ConfigurationRequestValid checks envelope placement. Operation payload schemas
// and typed handlers validate fields and revision semantics before mutation.
func ConfigurationRequestValid(req Request) bool {
	if !ConfigurationOperation(req.Operation) || req.JobID != "" || req.DurationMS != 0 || req.AfterGeneration != 0 || req.PublicationID != "" || req.SourcePath != "" || req.ArtifactKind != "" || req.LeaseID != "" || req.ReferenceID != "" || req.MaxBytes != 0 || len(req.Data) > MaxWorkPayload {
		return false
	}
	switch req.Operation {
	case "import-presets.list", "import-presets.create", "import-presets.show":
 return req.ItemID==""&&(req.Operation=="import-presets.list"||len(req.Data)>0)
 case "import-presets.update","import-presets.delete":return ValidID(req.ItemID)&&len(req.Data)>0
 case "pipelines.list", "speakers.list", "terms.list":
		return req.ItemID == ""
	case "terms.compile":
		return req.ItemID == ""
	case "pipelines.show", "speakers.show", "terms.show":
		return ValidID(req.ItemID) && len(req.Data) == 0
	case "pipelines.set", "speakers.set", "terms.set":
		return ValidID(req.ItemID) && len(req.Data) > 0
	default:
		return ValidID(req.ItemID)
	}
}
