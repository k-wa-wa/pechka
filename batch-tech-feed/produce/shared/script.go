package shared

import (
	"encoding/json"
	"fmt"
	"os"
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
	Image    string   `json:"image,omitempty"`
	Caption  string   `json:"caption,omitempty"`
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

func LoadScript(path string) (*Script, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read script %s: %w", path, err)
	}

	var script Script
	if err := json.Unmarshal(raw, &script); err != nil {
		return nil, fmt.Errorf("failed to parse script JSON: %w", err)
	}

	return &script, nil
}
