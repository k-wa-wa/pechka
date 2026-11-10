package shared_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/k-wa-wa/pechka/batch-tech-feed/produce/shared"
)

func intPtr(i int) *int {
	return &i
}

func TestBuildTimeline(t *testing.T) {
	script := &shared.Script{
		DigestDate: "2026-11-10",
		Title:      "Weekly Tech News",
		Sections: []shared.Section{
			{
				Seq: 1,
				Slide: shared.Slide{
					Layout:   "bullets",
					Title:    "Top Updates",
					Subtitle: "Overview",
					Items:    []string{"Item 1", "Item 2", "Item 3"},
				},
				Sources: []shared.SourceItem{
					{Publisher: "GitHub Blog", URL: "https://github.blog"},
					{URL: "https://openai.com/blog"}, // Fallback to URL when Publisher is empty
				},
				Narration: []shared.NarrationItem{
					{Text: "First update here.", Focus: intPtr(0)},
					{Text: "Second update follows.", Focus: intPtr(1)},
					{Text: "Third and final update.", Focus: intPtr(2)},
				},
			},
			{
				Seq: 2,
				Slide: shared.Slide{
					Layout: "title",
					Title:  "Conclusion",
				},
				Narration: []shared.NarrationItem{
					{Text: "That is all for today."},
				},
			},
		},
	}

	entries := shared.BuildTimeline(script)
	if len(entries) != 4 {
		t.Fatalf("expected 4 entries, got %d", len(entries))
	}

	// Entry 0 check
	e0 := entries[0]
	if e0.Seq != 0 || e0.SectionSeq != 1 || e0.Text != "First update here." {
		t.Errorf("unexpected entry 0: %+v", e0)
	}
	if e0.State.Revealed != 1 || e0.State.Highlight == nil || *e0.State.Highlight != 0 {
		t.Errorf("unexpected state 0: revealed=%d, highlight=%v", e0.State.Revealed, e0.State.Highlight)
	}
	if len(e0.State.Sources) != 2 || e0.State.Sources[0] != "GitHub Blog" || e0.State.Sources[1] != "https://openai.com/blog" {
		t.Errorf("unexpected sources: %v", e0.State.Sources)
	}

	// Entry 1 check
	e1 := entries[1]
	if e1.State.Revealed != 2 || e1.State.Highlight == nil || *e1.State.Highlight != 1 {
		t.Errorf("unexpected state 1: revealed=%d, highlight=%v", e1.State.Revealed, e1.State.Highlight)
	}

	// Entry 2 check
	e2 := entries[2]
	if e2.State.Revealed != 3 || e2.State.Highlight == nil || *e2.State.Highlight != 2 {
		t.Errorf("unexpected state 2: revealed=%d, highlight=%v", e2.State.Revealed, e2.State.Highlight)
	}

	// Entry 3 check (Section 2)
	e3 := entries[3]
	if e3.Seq != 3 || e3.SectionSeq != 2 || e3.Text != "That is all for today." {
		t.Errorf("unexpected entry 3: %+v", e3)
	}
	if e3.State.Layout != "title" || e3.State.Header != "Weekly Tech News" {
		t.Errorf("unexpected state 3: %+v", e3.State)
	}
}

func TestApplyDurations(t *testing.T) {
	entries := []*shared.Entry{
		{Seq: 0},
		{Seq: 1},
		{Seq: 2},
	}

	t.Run("duration count mismatch", func(t *testing.T) {
		_, err := shared.ApplyDurations(entries, []int{1000, 2000})
		if err == nil {
			t.Fatal("expected error on count mismatch, got nil")
		}
	})

	t.Run("successful calculation", func(t *testing.T) {
		durations := []int{1000, 2500, 1500}
		total, err := shared.ApplyDurations(entries, durations)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if total != 5000 {
			t.Errorf("expected total 5000, got %d", total)
		}

		if entries[0].StartMs != 0 || entries[0].EndMs != 1000 || entries[0].DurationMs() != 1000 {
			t.Errorf("unexpected entry 0 timing: %d -> %d", entries[0].StartMs, entries[0].EndMs)
		}
		if entries[1].StartMs != 1000 || entries[1].EndMs != 3500 || entries[1].DurationMs() != 2500 {
			t.Errorf("unexpected entry 1 timing: %d -> %d", entries[1].StartMs, entries[1].EndMs)
		}
		if entries[2].StartMs != 3500 || entries[2].EndMs != 5000 || entries[2].DurationMs() != 1500 {
			t.Errorf("unexpected entry 2 timing: %d -> %d", entries[2].StartMs, entries[2].EndMs)
		}
	})
}

func TestManifestSaveAndLoad(t *testing.T) {
	t.Run("empty entries results in zero TotalMs", func(t *testing.T) {
		script := &shared.Script{
			DigestDate: "2026-11-10",
			Title:      "Empty Demo",
		}
		manifest := shared.ToManifest(script, nil, 30, "/tmp/empty.wav")
		if manifest.TotalMs != 0 {
			t.Errorf("expected TotalMs = 0 for empty entries, got %d", manifest.TotalMs)
		}
	})

	t.Run("save and unmarshal full manifest", func(t *testing.T) {
		script := &shared.Script{
			DigestDate: "2026-11-10",
			Title:      "Demo Title",
		}
		entries := []*shared.Entry{
			{
				Seq:        0,
				SectionSeq: 1,
				Text:       "Narration line 1",
				StartMs:    0,
				EndMs:      3000,
				State: shared.SlideState{
					Layout: "bullets",
					Title:  "Slide 1",
				},
			},
		}

		manifest := shared.ToManifest(script, entries, 30, "/tmp/narration.wav")
		if manifest.DigestDate != "2026-11-10" || manifest.TotalMs != 3000 || manifest.FPS != 30 {
			t.Errorf("unexpected manifest content: %+v", manifest)
		}

		tmpDir := t.TempDir()
		outPath := filepath.Join(tmpDir, "manifest.json")
		if err := shared.SaveManifest(manifest, outPath); err != nil {
			t.Fatalf("failed to save manifest: %v", err)
		}

		data, err := os.ReadFile(outPath)
		if err != nil {
			t.Fatalf("failed to read saved manifest: %v", err)
		}

		var loaded shared.Manifest
		if err := json.Unmarshal(data, &loaded); err != nil {
			t.Fatalf("failed to unmarshal saved manifest: %v", err)
		}

		if loaded.DigestDate != manifest.DigestDate {
			t.Errorf("expected DigestDate %s, got %s", manifest.DigestDate, loaded.DigestDate)
		}
		if loaded.Title != manifest.Title {
			t.Errorf("expected Title %s, got %s", manifest.Title, loaded.Title)
		}
		if loaded.FPS != manifest.FPS {
			t.Errorf("expected FPS %d, got %d", manifest.FPS, loaded.FPS)
		}
		if loaded.TotalMs != manifest.TotalMs {
			t.Errorf("expected TotalMs %d, got %d", manifest.TotalMs, loaded.TotalMs)
		}
		if loaded.Narration != manifest.Narration {
			t.Errorf("expected Narration %s, got %s", manifest.Narration, loaded.Narration)
		}
		if len(loaded.Entries) != 1 {
			t.Fatalf("expected 1 entry, got %d", len(loaded.Entries))
		}
		if loaded.Entries[0].Text != "Narration line 1" || loaded.Entries[0].DurationMs() != 3000 {
			t.Errorf("unexpected entry data: %+v", loaded.Entries[0])
		}
	})
}
