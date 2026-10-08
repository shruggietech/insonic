// SPDX-License-Identifier: Apache-2.0
package contracts

func DesktopOperation(operation string) bool {
	switch operation {
	case "settings.show", "settings.set", "recordings.cues", "media.capture", "media.playback", "media.playback-check", "media.playback-close":
		return true
	}
	return false
}

func DesktopRequestValid(req Request) bool {
	if !DesktopOperation(req.Operation) || req.JobID != "" || req.DurationMS != 0 || req.AfterGeneration != 0 || req.PublicationID != "" || req.SourcePath != "" || req.ArtifactKind != "" || req.LeaseID != "" || req.ReferenceID != "" || req.MaxBytes != 0 || len(req.Data) > 1<<20 {
		return false
	}
	switch req.Operation {
	case "settings.show":
		return req.ItemID == "" && len(req.Data) == 0
	case "settings.set":
		return req.ItemID == "" && len(req.Data) > 0
	case "recordings.cues":
		return ValidID(req.ItemID)
	default:
		return ValidID(req.ItemID) && len(req.Data) > 0
	}
}
