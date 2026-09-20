package shared

import (
	"strings"
	"testing"
)

func TestValidateContent_OK(t *testing.T) {
	content := strings.Repeat("これは十分に長い本文テキストです。", 10)
	if err := ValidateContent(content); err != nil {
		t.Fatalf("expected valid content to pass, got error: %v", err)
	}
}

func TestValidateContent_TooShort(t *testing.T) {
	cases := []string{
		"",
		"   ",
		"短い本文",
	}
	for _, content := range cases {
		if err := ValidateContent(content); err == nil {
			t.Errorf("expected error for too-short content %q, got nil", content)
		}
	}
}

func TestValidateContent_SuspiciousPatterns(t *testing.T) {
	longEnough := strings.Repeat("x", MinContentLength)

	cases := []string{
		"bare-web-proxy によるエラーが発生しました。" + longEnough,
		"Checking your browser before accessing example.com. " + longEnough,
		"Just a moment... " + longEnough,
		"Please enable javascript and cookies to continue. " + longEnough,
		"Please verify you are a human before proceeding. " + longEnough,
		"Attention Required! | Cloudflare " + longEnough,
		"Access Denied - you don't have permission. " + longEnough,
		"403 Forbidden " + longEnough,
		"404 Not Found " + longEnough,
		"Sorry, this page not found. " + longEnough,
	}

	for _, content := range cases {
		if err := ValidateContent(content); err == nil {
			t.Errorf("expected suspicious content to be rejected: %q", content)
		}
	}
}

func TestValidateContent_TrimsWhitespaceBeforeLengthCheck(t *testing.T) {
	content := "\n\t  " + strings.Repeat("あ", MinContentLength-1) + "  \n"
	if err := ValidateContent(content); err == nil {
		t.Errorf("expected content just under the minimum length (after trim) to be rejected")
	}
}
