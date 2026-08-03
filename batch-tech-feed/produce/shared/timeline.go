package shared

import (
	"encoding/json"
	"fmt"
	"os"
)

// SlideState はスライドのある瞬間の状態。
type SlideState struct {
	Layout       string   `json:"layout"`
	Header       string   `json:"header"`
	Title        string   `json:"title"`
	Subtitle     string   `json:"subtitle"`
	Items        []string `json:"items"`
	Revealed     int      `json:"revealed"`
	Highlight    *int     `json:"highlight"`
	Code         string   `json:"code"`
	Language     string   `json:"language"`
	Image        string   `json:"image"`
	Caption      string   `json:"caption"`
	Diagram      string   `json:"diagram"`
	Sources      []string `json:"sources"`
	SectionSeq   int      `json:"section_seq"`
	SectionTotal int      `json:"section_total"`
}

// Entry はナレーション1文に対応する音声・スライド・字幕の単位。
type Entry struct {
	Seq        int        `json:"seq"`
	SectionSeq int        `json:"section_seq"`
	Text       string     `json:"text"`
	State      SlideState `json:"state"`
	AudioPath  string     `json:"audio"`
	StartMs    int        `json:"start_ms"`
	EndMs      int        `json:"end_ms"`
}

func (e *Entry) DurationMs() int {
	return e.EndMs - e.StartMs
}

func BuildTimeline(s *Script) []*Entry {
	total := len(s.Sections)
	header := s.Title

	var entries []*Entry
	seq := 0
	for _, sec := range s.Sections {
		states := statesForSection(sec, header, total)
		for i, line := range sec.Narration {
			state := states[i]
			entries = append(entries, &Entry{
				Seq:        seq,
				SectionSeq: sec.Seq,
				Text:       line.Text,
				State:      state,
			})
			seq++
		}
	}
	return entries
}

func statesForSection(sec Section, header string, total int) []SlideState {
	slide := sec.Slide
	var sources []string
	for _, src := range sec.Sources {
		val := src.Publisher
		if val == "" {
			val = src.URL
		}
		if val != "" {
			sources = append(sources, val)
		}
	}
	if sources == nil {
		sources = []string{}
	}

	items := slide.Items
	if items == nil {
		items = []string{}
	}

	base := SlideState{
		Layout:       slide.Layout,
		Header:       header,
		Title:        slide.Title,
		Subtitle:     slide.Subtitle,
		Items:        items,
		Code:         slide.Code,
		Language:     slide.Language,
		Image:        "",
		Caption:      "",
		Diagram:      slide.Diagram,
		Sources:      sources,
		SectionSeq:   sec.Seq,
		SectionTotal: total,
	}

	var states []SlideState
	revealed := 0
	for _, line := range sec.Narration {
		var highlight *int
		if slide.Layout == "bullets" && line.Focus != nil {
			idx := *line.Focus
			if idx+1 > revealed {
				revealed = idx + 1
			}
			highlight = line.Focus
		}

		st := base
		st.Revealed = revealed
		st.Highlight = highlight
		states = append(states, st)
	}

	return states
}

func ApplyDurations(entries []*Entry, durationsMs []int) (int, error) {
	if len(entries) != len(durationsMs) {
		return 0, fmt.Errorf("entry/duration count mismatch: %d entries, %d durations", len(entries), len(durationsMs))
	}

	cursor := 0
	for i, entry := range entries {
		dur := durationsMs[i]
		entry.StartMs = cursor
		cursor += dur
		entry.EndMs = cursor
	}
	return cursor, nil
}

type Manifest struct {
	DigestDate string   `json:"digest_date"`
	Title      string   `json:"title"`
	FPS        int      `json:"fps"`
	TotalMs    int      `json:"total_ms"`
	Narration  string   `json:"narration"`
	Entries    []*Entry `json:"entries"`
}

func ToManifest(s *Script, entries []*Entry, fps int, narrationPath string) *Manifest {
	totalMs := 0
	if len(entries) > 0 {
		totalMs = entries[len(entries)-1].EndMs
	}

	return &Manifest{
		DigestDate: s.DigestDate,
		Title:      s.Title,
		FPS:        fps,
		TotalMs:    totalMs,
		Narration:  narrationPath,
		Entries:    entries,
	}
}

func SaveManifest(m *Manifest, path string) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
