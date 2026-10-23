package domain

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

const (
	MimeTypeMarkdown = "text/markdown"
	MimeTypePDF      = "application/pdf"

	MaxMarkdownSizeBytes int64 = 10 * 1024 * 1024  // 10 MB
	MaxPDFSizeBytes      int64 = 100 * 1024 * 1024 // 100 MB
)

// DocumentFormatSpec は対応するドキュメント形式の仕様を定義する。
// 新しいドキュメント形式（.txt, .csv 等）を追加する場合は supportedDocumentFormats にエントリを追加する。
type DocumentFormatSpec struct {
	Extension       string
	MimeType        string
	StorageMimeType string
	StorageFilename string
	MaxSizeBytes    int64
	ValidateSniff   func(sample []byte) error
}

var markdownFormatSpec = DocumentFormatSpec{
	Extension:       ".md",
	MimeType:        MimeTypeMarkdown,
	StorageMimeType: "text/markdown; charset=utf-8",
	StorageFilename: "document.md",
	MaxSizeBytes:    MaxMarkdownSizeBytes,
	ValidateSniff:   validateUTF8TextSample,
}

var pdfFormatSpec = DocumentFormatSpec{
	Extension:       ".pdf",
	MimeType:        MimeTypePDF,
	StorageMimeType: "application/pdf",
	StorageFilename: "document.pdf",
	MaxSizeBytes:    MaxPDFSizeBytes,
	ValidateSniff:   validatePDFSample,
}

// supportedDocumentFormats は拡張子（小文字、ドット付き）をキーとする対応フォーマットレジストリ。
var supportedDocumentFormats = map[string]DocumentFormatSpec{
	".md":       markdownFormatSpec,
	".markdown": markdownFormatSpec,
	".pdf":      pdfFormatSpec,
}

// ResolveDocumentFormat はファイル名の拡張子から対応する DocumentFormatSpec を返す。
func ResolveDocumentFormat(filename string) (DocumentFormatSpec, bool) {
	ext := strings.ToLower(filepath.Ext(strings.TrimSpace(filename)))
	spec, ok := supportedDocumentFormats[ext]
	return spec, ok
}

// IsSupportedDocumentMimeType は DB 保存用の MIME タイプがサポート対象か判定する。
func IsSupportedDocumentMimeType(mimeType string) bool {
	normalized := strings.ToLower(strings.TrimSpace(strings.Split(mimeType, ";")[0]))
	for _, spec := range supportedDocumentFormats {
		if spec.MimeType == normalized {
			return true
		}
	}
	return false
}

// DocumentObjectKey は MinIO 上の正規化された保存先キーを返す。
// ユーザー入力の元ファイル名をそのままパスに含めず、固定名(document.md / document.pdf)に正規化する。
func DocumentObjectKey(shortID string, spec DocumentFormatSpec) string {
	return fmt.Sprintf("resources/documents/%s/%s", shortID, spec.StorageFilename)
}

// ValidateDocumentUpload はファイルサイズおよび先頭バイト列(マジックバイト / UTF-8)を検証する。
func ValidateDocumentUpload(spec DocumentFormatSpec, size int64, sample []byte) error {
	if size <= 0 {
		return fmt.Errorf("document file is empty")
	}
	if spec.MaxSizeBytes > 0 && size > spec.MaxSizeBytes {
		return fmt.Errorf("document file exceeds max size (%d bytes)", spec.MaxSizeBytes)
	}
	if spec.ValidateSniff != nil {
		if err := spec.ValidateSniff(sample); err != nil {
			return err
		}
	}
	return nil
}

func validatePDFSample(sample []byte) error {
	if !bytes.HasPrefix(sample, []byte("%PDF-")) {
		return fmt.Errorf("invalid PDF file: missing %%PDF- header")
	}
	return nil
}

func validateUTF8TextSample(sample []byte) error {
	if len(sample) == 0 {
		return fmt.Errorf("document file is empty")
	}
	if bytes.IndexByte(sample, 0) != -1 {
		return fmt.Errorf("invalid markdown file: binary content detected")
	}
	// 先頭サンプル切り出し時に末尾のマルチバイト文字が途中で切れている場合を考慮し、
	// 末尾最大3バイトの不完全なUTF-8シーケンスを除外して判定する。
	trimmed := trimIncompleteUTF8Tail(sample)
	if !utf8.Valid(trimmed) {
		return fmt.Errorf("invalid markdown file: must be UTF-8 encoded text")
	}
	return nil
}

func trimIncompleteUTF8Tail(b []byte) []byte {
	for i := 1; i <= 3 && len(b) >= i; i++ {
		if utf8.Valid(b) {
			return b
		}
		b = b[:len(b)-1]
	}
	return b
}
