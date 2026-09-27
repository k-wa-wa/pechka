package handler

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"

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
	minioClient minioPutObjecter
	bucket      string
	dynClient   dynamic.Interface
	snowflake   *snowflake.Node
}

func NewUploadHandler(contentRepo contentCreator, minioClient minioPutObjecter, bucket string, dynClient dynamic.Interface, node *snowflake.Node) *UploadHandler {
	return &UploadHandler{
		contentRepo: contentRepo,
		minioClient: minioClient,
		bucket:      bucket,
		dynClient:   dynClient,
		snowflake:   node,
	}
}

type UploadVideoResponse struct {
	Message      string          `json:"message"`
	WorkflowName string          `json:"workflow_name"`
	Content      *domain.Content `json:"content"`
}

// UploadVideo は管理画面から送られた動画ファイルを MinIO の一時領域に保存し、
// ETL 側の Argo Workflow (アップロード用エントリーポイント) を起動する。
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

	src, err := fileHeader.Open()
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "failed to read uploaded file")
	}
	defer src.Close()

	ctx := c.Request().Context()
	shortID := h.snowflake.Generate().String()
	filename := filepath.Base(fileHeader.Filename)
	objectKey := fmt.Sprintf("uploads/%s/%s", shortID, filename)

	if _, err := h.minioClient.PutObject(ctx, h.bucket, objectKey, src, fileHeader.Size, minio.PutObjectOptions{
		ContentType: fileHeader.Header.Get(echo.HeaderContentType),
	}); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("failed to upload file to storage: %v", err))
	}

	content, err := h.contentRepo.Create(ctx, pgRepo.CreateContentParams{
		ShortID:     shortID,
		ContentType: domain.ContentTypeVideo,
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
				"namespace":    "pechka",
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

	created, err := h.dynClient.Resource(workflowGVR).Namespace("pechka").Create(ctx, wf, metav1.CreateOptions{})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("failed to trigger ETL workflow: %v", err))
	}

	return c.JSON(http.StatusAccepted, UploadVideoResponse{
		Message:      "Video uploaded and ingest workflow successfully submitted",
		WorkflowName: created.GetName(),
		Content:      content,
	})
}
