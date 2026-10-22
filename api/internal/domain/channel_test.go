package domain_test

import (
	"strings"
	"testing"

	"github.com/k-wa-wa/pechka/api/internal/domain"
)

func TestIsValidChannelID(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{name: "valid bluray", input: "bluray", want: true},
		{name: "valid tech-feed", input: "tech-feed", want: true},
		{name: "valid single char", input: "a", want: true},
		{name: "valid single digit", input: "1", want: true},
		{name: "valid alphanumeric with hyphen", input: "ch-01-archive", want: true},
		{name: "valid max length 50", input: "a" + strings.Repeat("b", 49), want: true},
		{name: "invalid empty", input: "", want: false},
		{name: "invalid length 51", input: "a" + strings.Repeat("b", 50), want: false},
		{name: "invalid leading hyphen", input: "-bluray", want: false},
		{name: "invalid uppercase", input: "Tech-Feed", want: false},
		{name: "invalid underscore", input: "tech_feed", want: false},
		{name: "invalid space", input: "tech feed", want: false},
		{name: "invalid slash", input: "tech/feed", want: false},
		{name: "invalid sql injection payload", input: "bluray' OR '1'='1", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := domain.IsValidChannelID(tt.input); got != tt.want {
				t.Errorf("IsValidChannelID(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
