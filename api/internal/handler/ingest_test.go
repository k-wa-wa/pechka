package handler_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	k8stesting "k8s.io/client-go/testing"

	"github.com/k-wa-wa/pechka/api/internal/domain"
	"github.com/k-wa-wa/pechka/api/internal/handler"
)

func TestIngestHandler_TriggerIngest_MissingDiscLabel(t *testing.T) {
	h := handler.NewIngestHandler(newFakeDynamicClient(), "pechka")
	e := echo.New()

	req := httptest.NewRequest(http.MethodPost, "/v1/contents/ingest", strings.NewReader(`{"content_title":"Title"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set(domain.HeaderChannelID, "bluray")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.TriggerIngest(c)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	he := err.(*echo.HTTPError)
	if he.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", he.Code)
	}
}

func TestIngestHandler_TriggerIngest_MissingChannelId(t *testing.T) {
	h := handler.NewIngestHandler(newFakeDynamicClient(), "pechka")
	e := echo.New()

	req := httptest.NewRequest(http.MethodPost, "/v1/contents/ingest", strings.NewReader(`{"disc_label":"DISC_001","content_title":"Title"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.TriggerIngest(c)
	if err == nil {
		t.Fatal("expected error when channel_id is missing, got nil")
	}
	he := err.(*echo.HTTPError)
	if he.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", he.Code)
	}
}

func TestIngestHandler_TriggerIngest_UnknownChannelId(t *testing.T) {
	channelRepo := &mockPgChannelRepo{
		existsFn: func(_ context.Context, id string) (bool, error) {
			return id == "bluray" || id == "tech-feed", nil
		},
	}
	h := handler.NewIngestHandler(newFakeDynamicClient(), "pechka").WithChannelRepository(channelRepo)
	e := echo.New()

	req := httptest.NewRequest(http.MethodPost, "/v1/contents/ingest", strings.NewReader(`{"disc_label":"DISC_001","content_title":"Title"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set(domain.HeaderChannelID, "unknown-ch")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.TriggerIngest(c)
	if err == nil {
		t.Fatal("expected error for unknown channel, got nil")
	}
	he := err.(*echo.HTTPError)
	if he.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", he.Code)
	}
}

func TestIngestHandler_TriggerIngest_SuccessWithHeader(t *testing.T) {
	var createdWf *unstructured.Unstructured
	dynClient := newFakeDynamicClient()
	dynClient.PrependReactor("create", "workflows", func(action k8stesting.Action) (bool, runtime.Object, error) {
		createAction := action.(k8stesting.CreateAction)
		obj := createAction.GetObject().(*unstructured.Unstructured).DeepCopy()
		obj.SetName("etl-bluray-manual-xyz")
		createdWf = obj
		return true, obj, nil
	})

	channelRepo := &mockPgChannelRepo{
		existsFn: func(_ context.Context, id string) (bool, error) {
			return id == "bluray", nil
		},
	}
	h := handler.NewIngestHandler(dynClient, "pechka").WithChannelRepository(channelRepo)
	e := echo.New()

	req := httptest.NewRequest(http.MethodPost, "/v1/contents/ingest", strings.NewReader(`{"disc_label":"DISC_001","content_title":"Movie 1"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set(domain.HeaderChannelID, "bluray")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := h.TriggerIngest(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", rec.Code)
	}

	var resp handler.IngestResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.WorkflowName != "etl-bluray-manual-xyz" {
		t.Errorf("expected workflow_name etl-bluray-manual-xyz, got %q", resp.WorkflowName)
	}

	if createdWf == nil {
		t.Fatal("expected workflow to be created")
	}
	params, found, err := unstructured.NestedSlice(createdWf.Object, "spec", "arguments", "parameters")
	if err != nil || !found {
		t.Fatalf("expected workflow parameters, found=%v err=%v", found, err)
	}
	var gotChannelParam string
	for _, p := range params {
		pm, ok := p.(map[string]interface{})
		if ok && pm["name"] == "channel-id" {
			gotChannelParam, _ = pm["value"].(string)
		}
	}
	if gotChannelParam != "bluray" {
		t.Errorf("expected workflow channel-id parameter 'bluray', got %q", gotChannelParam)
	}
}

func TestIngestHandler_TriggerIngest_WorkflowFailure(t *testing.T) {
	dynClient := newFakeDynamicClient()
	dynClient.PrependReactor("create", "workflows", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("k8s api error")
	})

	h := handler.NewIngestHandler(dynClient, "pechka")
	e := echo.New()

	req := httptest.NewRequest(http.MethodPost, "/v1/contents/ingest", strings.NewReader(`{"disc_label":"DISC_001","channel_id":"bluray"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.TriggerIngest(c)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	he := err.(*echo.HTTPError)
	if he.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", he.Code)
	}
}
