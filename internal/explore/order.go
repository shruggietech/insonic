// SPDX-License-Identifier: Apache-2.0
package explore

func mapOrderField(field string) string {
	switch field {
	case "media.originated_at":
		return "recording_date"
	case "media.id", "speaker.id", "model.id":
		return "id"
	case "cue.start_us":
		return "source_start_us"
	case "score":
		return "relevance"
	}
	return field
}
