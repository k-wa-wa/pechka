package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/minio/minio-go/v7"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8stesting "k8s.io/client-go/testing"

	fakedynamic "k8s.io/client-go/dynamic/fake"

	"github.com/k-wa-wa/pechka/api/internal/domain"
	"github.com/k-wa-wa/pechka/api/internal/handler"
	pgRepo "github.com/k-wa-wa/pechka/api/internal/repository/postgres"
)

type mockContentCreator struct {
	createFn func(ctx context.Context, params pgRepo.CreateContentParams) (*domain.Content, error)
}

func (m *mockContentCreator) Create(ctx context.Context, params pgRepo.CreateContentParams) (*domain.Content, error) {
	return m.createFn(ctx, params)
}

type mockMinioPutObjecter struct {
	putObjectFn func(ctx context.Context, bucketName, objectName string, reader io.Reader, objectSize int64, opts minio.PutObjectOptions) (minio.UploadInfo, error)
}

func (m *mockMinioPutObjecter) PutObject(ctx context.Context, bucketName, objectName string, reader io.Reader, objectSize int64, opts minio.PutObjectOptions) (minio.UploadInfo, error) {
	return m.putObjectFn(ctx, bucketName, objectName, reader, objectSize, opts)
}

var workflowGVR = schema.GroupVersionResource{Group: "argoproj.io", Version: "v1alpha1", Resource: "workflows"}

func newFakeDynamicClient() *fakedynamic.FakeDynamicClient {
	scheme := runtime.NewScheme()
	listKinds := map[schema.GroupVersionResource]string{workflowGVR: "WorkflowList"}
	client := fakedynamic.NewSimpleDynamicClientWithCustomListKinds(scheme, listKinds)
	// フェイクの ObjectTracker は generateName を展開しないため、
	// Create をラップして固定名を付与する reactor を差し込む。
	client.PrependReactor("create", "workflows", func(action k8stesting.Action) (bool, runtime.Object, error) {
		createAction := action.(k8stesting.CreateAction)
		obj := createAction.GetObject().(*unstructured.Unstructured).DeepCopy()
		obj.SetName("etl-upload-manual-abc123")
		return true, obj, nil
	})
	return client
}

func newMultipartUploadRequest(t *testing.T, fields map[string]string, fileFieldPresent bool) (*http.Request, string) {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	for k, v := range fields {
		if err := writer.WriteField(k, v); err != nil {
			t.Fatalf("write field: %v", err)
		}
	}
	if fileFieldPresent {
		part, err := writer.CreateFormFile("file", "movie.mp4")
		if err != nil {
			t.Fatalf("create form file: %v", err)
		}
		if _, err := part.Write([]byte("fake video bytes")); err != nil {
			t.Fatalf("write file content: %v", err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/admin/contents/upload", body)
	req.Header.Set(echo.HeaderContentType, writer.FormDataContentType())
	return req, writer.FormDataContentType()
}

func TestUploadHandler_UploadVideo_MissingFile(t *testing.T) {
	h := handler.NewUploadHandler(&mockContentCreator{}, &mockMinioPutObjecter{}, "pechka", newFakeDynamicClient(), newSnowflakeNode(t))
	e := echo.New()

	req, _ := newMultipartUploadRequest(t, map[string]string{"title": "Test Video"}, false)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.UploadVideo(c)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	he := err.(*echo.HTTPError)
	if he.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", he.Code)
	}
}

func TestUploadHandler_UploadVideo_MissingTitle(t *testing.T) {
	h := handler.NewUploadHandler(&mockContentCreator{}, &mockMinioPutObjecter{}, "pechka", newFakeDynamicClient(), newSnowflakeNode(t))
	e := echo.New()

	req, _ := newMultipartUploadRequest(t, map[string]string{}, true)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.UploadVideo(c)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	he := err.(*echo.HTTPError)
	if he.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", he.Code)
	}
}

func TestUploadHandler_UploadVideo_MinioFailure(t *testing.T) {
	contentRepo := &mockContentCreator{
		createFn: func(ctx context.Context, params pgRepo.CreateContentParams) (*domain.Content, error) {
			t.Fatal("content repo Create should not be called when minio upload fails")
			return nil, nil
		},
	}
	minioClient := &mockMinioPutObjecter{
		putObjectFn: func(ctx context.Context, bucketName, objectName string, reader io.Reader, objectSize int64, opts minio.PutObjectOptions) (minio.UploadInfo, error) {
			return minio.UploadInfo{}, errors.New("connection refused")
		},
	}
	h := handler.NewUploadHandler(contentRepo, minioClient, "pechka", newFakeDynamicClient(), newSnowflakeNode(t))
	e := echo.New()

	req, _ := newMultipartUploadRequest(t, map[string]string{"title": "Test Video"}, true)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.UploadVideo(c)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	he := err.(*echo.HTTPError)
	if he.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", he.Code)
	}
}

func TestUploadHandler_UploadVideo_Success(t *testing.T) {
	var gotObjectKey string
	var gotParams pgRepo.CreateContentParams

	minioClient := &mockMinioPutObjecter{
		putObjectFn: func(ctx context.Context, bucketName, objectName string, reader io.Reader, objectSize int64, opts minio.PutObjectOptions) (minio.UploadInfo, error) {
			gotObjectKey = objectName
			if bucketName != "pechka" {
				t.Errorf("expected bucket pechka, got %s", bucketName)
			}
			return minio.UploadInfo{}, nil
		},
	}
	contentRepo := &mockContentCreator{
		createFn: func(ctx context.Context, params pgRepo.CreateContentParams) (*domain.Content, error) {
			gotParams = params
			return &domain.Content{
				ID:          "1",
				ShortID:     params.ShortID,
				ContentType: params.ContentType,
				Title:       params.Title,
				Description: params.Description,
				Tags:        params.Tags,
				Status:      params.Status,
			}, nil
		},
	}

	h := handler.NewUploadHandler(contentRepo, minioClient, "pechka", newFakeDynamicClient(), newSnowflakeNode(t))
	e := echo.New()

	req, _ := newMultipartUploadRequest(t, map[string]string{
		"title":       "Test Video",
		"description": "a video",
		"tags":        "foo, bar",
	}, true)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := h.UploadVideo(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", rec.Code)
	}

	if gotParams.ContentType != domain.ContentTypeVideo {
		t.Errorf("expected content_type video, got %s", gotParams.ContentType)
	}
	if gotParams.DiscID != nil {
		t.Errorf("expected disc_id nil, got %v", *gotParams.DiscID)
	}
	if gotParams.Status != domain.ContentStatusPending {
		t.Errorf("expected status pending, got %s", gotParams.Status)
	}
	if len(gotParams.Tags) != 2 || gotParams.Tags[0] != "foo" || gotParams.Tags[1] != "bar" {
		t.Errorf("expected tags [foo bar], got %v", gotParams.Tags)
	}
	if gotObjectKey != "uploads/"+gotParams.ShortID+"/movie.mp4" {
		t.Errorf("unexpected object key: %s", gotObjectKey)
	}

	var resp handler.UploadVideoResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.WorkflowName == "" {
		t.Errorf("expected non-empty workflow name")
	}
	if resp.Content == nil || resp.Content.Title != "Test Video" {
		t.Errorf("unexpected content in response: %+v", resp.Content)
	}
}

func TestUploadHandler_UploadVideo_WorkflowTriggerFailure(t *testing.T) {
	minioClient := &mockMinioPutObjecter{
		putObjectFn: func(ctx context.Context, bucketName, objectName string, reader io.Reader, objectSize int64, opts minio.PutObjectOptions) (minio.UploadInfo, error) {
			return minio.UploadInfo{}, nil
		},
	}
	contentRepo := &mockContentCreator{
		createFn: func(ctx context.Context, params pgRepo.CreateContentParams) (*domain.Content, error) {
			return &domain.Content{ID: "1", ShortID: params.ShortID, Title: params.Title, Status: params.Status}, nil
		},
	}

	dynClient := newFakeDynamicClient()
	dynClient.PrependReactor("create", "workflows", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("workflow controller unavailable")
	})

	h := handler.NewUploadHandler(contentRepo, minioClient, "pechka", dynClient, newSnowflakeNode(t))
	e := echo.New()

	req, _ := newMultipartUploadRequest(t, map[string]string{"title": "Test Video"}, true)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.UploadVideo(c)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	he := err.(*echo.HTTPError)
	if he.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", he.Code)
	}
}
