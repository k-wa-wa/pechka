package shared

import (
	"fmt"
	"strings"
)

// Script は動画台本全体のデータ構造。
type Script struct {
	DigestDate string    `json:"digest_date"`
	Title      string    `json:"title"`
	Sections   []Section `json:"sections"`
}

// Section は1つのスライド・トピックに対応するセクション。
type Section struct {
	Seq       int             `json:"seq"`
	Slide     Slide           `json:"slide"`
	Narration []NarrationItem `json:"narration"`
	Sources   []SourceItem    `json:"sources,omitempty"`
}

// Slide はスライドの表示要素。
type Slide struct {
	Layout   string   `json:"layout"` // title | bullets | code | diagram
	Title    string   `json:"title"`
	Subtitle string   `json:"subtitle,omitempty"`
	Items    []string `json:"items,omitempty"`
	Code     string   `json:"code,omitempty"`
	Language string   `json:"language,omitempty"`
	Diagram  string   `json:"diagram,omitempty"`
}

// NarrationItem はナレーションの1文に対応する読み上げ単位。
type NarrationItem struct {
	Text  string `json:"text"`
	Focus *int   `json:"focus"`
}

// SourceItem は参照元情報。
type SourceItem struct {
	Title     string `json:"title"`
	URL       string `json:"url"`
	Publisher string `json:"publisher"`
}

// ValidateScript は生成された台本の構造・制約を検証する。
func ValidateScript(s *Script, allowedURLs map[string]bool) error {
	if s.DigestDate == "" {
		return fmt.Errorf("digest_date is required")
	}
	if s.Title == "" {
		return fmt.Errorf("title is required")
	}
	if len(s.Sections) < 2 {
		return fmt.Errorf("sections must contain at least a title section and topic sections (got %d)", len(s.Sections))
	}

	for i, sec := range s.Sections {
		secIdx := i + 1
		if sec.Seq != secIdx {
			// seq補正または検証
		}

		if i == 0 {
			if sec.Slide.Layout != "title" {
				return fmt.Errorf("section 1 must have layout='title' (got %s)", sec.Slide.Layout)
			}
		} else {
			if sec.Slide.Layout == "title" {
				return fmt.Errorf("section %d cannot have layout='title'", secIdx)
			}
		}

		switch sec.Slide.Layout {
		case "title", "bullets", "code", "diagram":
			// 有効なレイアウト
		default:
			return fmt.Errorf("section %d: invalid layout '%s'", secIdx, sec.Slide.Layout)
		}

		if sec.Slide.Layout == "bullets" && len(sec.Slide.Items) == 0 {
			return fmt.Errorf("section %d: layout='bullets' requires items", secIdx)
		}

		if len(sec.Narration) == 0 {
			return fmt.Errorf("section %d: narration cannot be empty", secIdx)
		}

		for j, narr := range sec.Narration {
			if strings.TrimSpace(narr.Text) == "" {
				return fmt.Errorf("section %d narration %d: text is empty", secIdx, j+1)
			}
			if narr.Focus != nil {
				if sec.Slide.Layout != "bullets" {
					return fmt.Errorf("section %d narration %d: focus is only allowed for 'bullets' layout", secIdx, j+1)
				}
				idx := *narr.Focus
				if idx < 0 || idx >= len(sec.Slide.Items) {
					return fmt.Errorf("section %d narration %d: focus index %d out of bounds (items len=%d)", secIdx, j+1, idx, len(sec.Slide.Items))
				}
			}
		}

		if i > 0 && allowedURLs != nil {
			for _, src := range sec.Sources {
				if src.URL != "" && !allowedURLs[src.URL] {
					return fmt.Errorf("section %d: source URL '%s' is not in verified primary URLs list", secIdx, src.URL)
				}
			}
		}
	}

	return nil
}
