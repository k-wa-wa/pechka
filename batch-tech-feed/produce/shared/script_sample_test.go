package shared

import "testing"

// 見本の台本 (examples/script_sample.json) は preview の produce の Job が読む。
// 台本の型が変わったときに見本が置き去りにならないよう、読み込めることを確かめる。
func TestLoadScriptSample(t *testing.T) {
	s, err := LoadScript("../examples/script_sample.json")
	if err != nil {
		t.Fatalf("見本の台本を読み込めない: %v", err)
	}
	if len(s.Sections) == 0 {
		t.Fatal("見本の台本にセクションが無い")
	}
	for _, sec := range s.Sections {
		for _, n := range sec.Narration {
			if n.Text == "" {
				t.Fatalf("セクション %d に空のナレーションがある", sec.Seq)
			}
		}
	}
}
