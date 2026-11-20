package postgres

import (
	"strings"
	"testing"
)

func TestSubtitleQueriesExcludeDeletedContents(t *testing.T) {
	queries := map[string]string{
		"listTracksQuery":           listTracksQuery,
		"getTrackQuery":             getTrackQuery,
		"setTrackStatusQuery":       setTrackStatusQuery,
		"listCuesQuery":             listCuesQuery,
		"insertCueCheckParentQuery": insertCueCheckParentQuery,
		"deleteCueQuery":            deleteCueQuery,
	}
	for name, q := range queries {
		if !strings.Contains(q, "c.deleted_at IS NULL") {
			t.Errorf("%s must filter c.deleted_at IS NULL, got %q", name, q)
		}
	}

	text := "updated"
	qUpdate, args := buildUpdateCueQuery(UpdateCueParams{ID: "cue-1", Text: &text})
	if !strings.Contains(qUpdate, "c.deleted_at IS NULL") {
		t.Errorf("buildUpdateCueQuery must filter c.deleted_at IS NULL, got %q", qUpdate)
	}
	if len(args) != 2 || args[0] != "updated" || args[1] != "cue-1" {
		t.Errorf("unexpected update cue args: %v", args)
	}
}
