package handler

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/labstack/echo/v4"
	"github.com/minio/minio-go/v7"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	"github.com/k-wa-wa/pechka/api/internal/domain"
	pgRepo "github.com/k-wa-wa/pechka/api/internal/repository/postgres"
)

type contentCreator interface {
	Create(ctx context.Context, params pgRepo.CreateContentParams) (*domain.Content, error)
}

// minioPutObjecter は minio.Client.PutObject と同じシグネチャを持つ最小インターフェース。
// テストではモックに差し替える。
type minioPutObjecter interface {
	PutObject(ctx context.Context, bucketName, objectName string, reader io.Reader, objectSize int64, opts minio.PutObjectOptions) (minio.UploadInfo, error)
}

type UploadHandler struct {
	contentRepo contentCreator
	channelRepo pgChannelRepository
	minioClient minioPutObjecter
	bucket      string
	dynClient   dynamic.Interface
	snowflake   *snowflake.Node
	namespace   string
}

func NewUploadHandler(contentRepo contentCreator, minioClient minioPutObjecter, bucket string, dynClient dynamic.Interface, node *snowflake.Node, namespace string) *UploadHandler {
	return &UploadHandler{
		contentRepo: contentRepo,
		minioClient: minioClient,
		bucket:      bucket,
		dynClient:   dynClient,
		snowflake:   node,
		namespace:   namespace,
	}
}

func (h *UploadHandler) WithChannelRepository(channelRepo pgChannelRepository) *UploadHandler {
	h.channelRepo = channelRepo
	return h
}

type UploadVideoResponse struct {
	Message      string          `json:"message"`
	WorkflowName string          `json:"workflow_name"`
	Content      *domain.Content `json:"content"`
}

// UploadVideo は管理画面またはAPIから送られた動画・ドキュメント(.md/.pdf)を受け付ける。
// - ドキュメント(.md/.pdf): フォーマット・マジックバイト検証の上で MinIO の公開領域へ直接保存し、status=ready で即時公開する。
// - 動画: MinIO の一時領域に保存し、ETL 側の Argo Workflow (アップロード用エントリーポイント) を起動する。
func (h *UploadHandler) UploadVideo(c echo.Context) error {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "file is required")
	}

	title := strings.TrimSpace(c.FormValue("title"))
	if title == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "title is required")
	}
	description := c.FormValue("description")

	var tags []string
	if tagsParam := c.FormValue("tags"); tagsParam != "" {
		for _, t := range strings.Split(tagsParam, ",") {
			if t = strings.TrimSpace(t); t != "" {
				tags = append(tags, t)
			}
		}
	}
	if tags == nil {
		tags = []string{}
	}

	requestedType := domain.ContentType(strings.TrimSpace(c.FormValue("content_type")))
	docSpec, isDocExt := domain.ResolveDocumentFormat(fileHeader.Filename)
	if requestedType == domain.ContentTypeDocument && !isDocExt {
		return echo.NewHTTPError(http.StatusBadRequest, "unsupported document format: only .md and .pdf are supported")
	}
	if requestedType != "" && requestedType != domain.ContentTypeVideo && requestedType != domain.ContentTypeDocument {
		return echo.NewHTTPError(http.StatusBadRequest, "unsupported content_type")
	}

	src, err := fileHeader.Open()
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "failed to read uploaded file")
	}
	defer src.Close()

	// TODO(security): Integrate with an antivirus API / CDR tool to scan and strip active content from uploaded PDFs.
	var uploadReader io.Reader = src
	if isDocExt {
		sniffBuf := make([]byte, 512)
		n, readErr := io.ReadFull(src, sniffBuf)
		if readErr != nil && readErr != io.EOF && readErr != io.ErrUnexpectedEOF {
			return echo.NewHTTPError(http.StatusBadRequest, "failed to inspect uploaded file")
		}
		sample := sniffBuf[:n]
		if err := domain.ValidateDocumentUpload(docSpec, fileHeader.Size, sample); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		uploadReader = io.MultiReader(bytes.NewReader(sample), src)
	}

	ctx := c.Request().Context()
	var formChannel *string
	if ch := strings.TrimSpace(c.FormValue("channel_id")); ch != "" {
		formChannel = &ch
	}
	channelID, err := resolveChannelID(ctx, c, formChannel, h.channelRepo)
	if err != nil {
		return err
	}

	shortID := h.snowflake.Generate().String()

	if isDocExt {
		objectKey := domain.DocumentObjectKey(shortID, docSpec)
		if _, err := h.minioClient.PutObject(ctx, h.bucket, objectKey, uploadReader, fileHeader.Size, minio.PutObjectOptions{
			ContentType: docSpec.StorageMimeType,
		}); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("failed to upload document to storage: %v", err))
		}

		mimeType := docSpec.MimeType
		now := time.Now().UTC()
		content, err := h.contentRepo.Create(ctx, pgRepo.CreateContentParams{
			ShortID:          shortID,
			ContentType:      domain.ContentTypeDocument,
			ChannelID:        channelID,
			DiscID:           nil,
			Title:            title,
			Description:      description,
			Tags:             tags,
			Status:           domain.ContentStatusReady,
			DocumentKey:      &objectKey,
			DocumentMimeType: &mimeType,
			PublishedAt:      &now,
		})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}

		return c.JSON(http.StatusCreated, UploadVideoResponse{
			Message:      "Document uploaded and published",
			WorkflowName: "",
			Content:      content,
		})
	}

	filename := filepath.Base(fileHeader.Filename)
	objectKey := fmt.Sprintf("uploads/%s/%s", shortID, filename)

	if _, err := h.minioClient.PutObject(ctx, h.bucket, objectKey, uploadReader, fileHeader.Size, minio.PutObjectOptions{
		ContentType: fileHeader.Header.Get(echo.HeaderContentType),
	}); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("failed to upload file to storage: %v", err))
	}

	content, err := h.contentRepo.Create(ctx, pgRepo.CreateContentParams{
		ShortID:     shortID,
		ContentType: domain.ContentTypeVideo,
		ChannelID:   channelID,
		DiscID:      nil,
		Title:       title,
		Description: description,
		Tags:        tags,
		Status:      domain.ContentStatusPending,
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	workflowGVR := schema.GroupVersionResource{
		Group:    "argoproj.io",
		Version:  "v1alpha1",
		Resource: "workflows",
	}

	wf := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "argoproj.io/v1alpha1",
			"kind":       "Workflow",
			"metadata": map[string]interface{}{
				"generateName": "etl-upload-manual-",
				"namespace":    h.namespace,
			},
			"spec": map[string]interface{}{
				"workflowTemplateRef": map[string]interface{}{
					"name": "etl-bluray",
				},
				"entrypoint": "upload",
				"arguments": map[string]interface{}{
					"parameters": []interface{}{
						map[string]interface{}{
							"name":  "object-key",
							"value": objectKey,
						},
						map[string]interface{}{
							"name":  "content-title",
							"value": title,
						},
						map[string]interface{}{
							"name":  "short-id",
							"value": shortID,
						},
						map[string]interface{}{
							"name":  "content-id",
							"value": content.ID,
						},
					},
				},
			},
		},
	}

	slog.Info("triggering ETL workflow", "namespace", h.namespace)
	created, err := h.dynClient.Resource(workflowGVR).Namespace(h.namespace).Create(ctx, wf, metav1.CreateOptions{})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("failed to trigger ETL workflow: %v", err))
	}

	return c.JSON(http.StatusAccepted, UploadVideoResponse{
		Message:      "Video uploaded and ingest workflow successfully submitted",
		WorkflowName: created.GetName(),
		Content:      content,
	})
}
