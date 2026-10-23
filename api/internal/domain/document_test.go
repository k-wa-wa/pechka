package domain_test

import (
	"testing"

	"github.com/k-wa-wa/pechka/api/internal/domain"
)

func TestResolveDocumentFormat(t *testing.T) {
	tests := []struct {
		filename string
		wantOk   bool
		wantMime string
		wantFile string
	}{
		{"guide.md", true, domain.MimeTypeMarkdown, "document.md"},
		{"README.MARKDOWN", true, domain.MimeTypeMarkdown, "document.md"},
		{"paper.pdf", true, domain.MimeTypePDF, "document.pdf"},
		{"movie.mp4", false, "", ""},
		{"archive.zip", false, "", ""},
	}

	for _, tt := range tests {
		spec, ok := domain.ResolveDocumentFormat(tt.filename)
		if ok != tt.wantOk {
			t.Errorf("ResolveDocumentFormat(%q) ok = %v, want %v", tt.filename, ok, tt.wantOk)
			continue
		}
		if !ok {
			continue
		}
		if spec.MimeType != tt.wantMime {
			t.Errorf("ResolveDocumentFormat(%q) MimeType = %q, want %q", tt.filename, spec.MimeType, tt.wantMime)
		}
		if spec.StorageFilename != tt.wantFile {
			t.Errorf("ResolveDocumentFormat(%q) StorageFilename = %q, want %q", tt.filename, spec.StorageFilename, tt.wantFile)
		}
	}
}

func TestValidateDocumentUpload(t *testing.T) {
	mdSpec, _ := domain.ResolveDocumentFormat("note.md")
	pdfSpec, _ := domain.ResolveDocumentFormat("slide.pdf")

	if err := domain.ValidateDocumentUpload(mdSpec, 24, []byte("# タイトル\n\n本文テキスト")); err != nil {
		t.Errorf("expected valid markdown, got error: %v", err)
	}

	if err := domain.ValidateDocumentUpload(mdSpec, 10, []byte{'h', 'e', 'l', 0, 'o'}); err == nil {
		t.Error("expected error for binary null byte in markdown, got nil")
	}

	if err := domain.ValidateDocumentUpload(pdfSpec, 16, []byte("%PDF-1.7\n%stub\n")); err != nil {
		t.Errorf("expected valid PDF, got error: %v", err)
	}

	if err := domain.ValidateDocumentUpload(pdfSpec, 16, []byte("not a pdf file")); err == nil {
		t.Error("expected error for invalid PDF magic bytes, got nil")
	}

	if err := domain.ValidateDocumentUpload(mdSpec, domain.MaxMarkdownSizeBytes+1, []byte("# too big")); err == nil {
		t.Error("expected error for oversized markdown file, got nil")
	}
}
