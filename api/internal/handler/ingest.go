package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

type IngestHandler struct {
	dynClient dynamic.Interface
}

func NewIngestHandler(dynClient dynamic.Interface) *IngestHandler {
	return &IngestHandler{
		dynClient: dynClient,
	}
}

type IngestRequest struct {
	DiscLabel    string `json:"disc_label"`
	ContentTitle string `json:"content_title"`
}

type IngestResponse struct {
	Message      string `json:"message"`
	WorkflowName string `json:"workflow_name"`
}

func (h *IngestHandler) TriggerIngest(c echo.Context) error {
	var req IngestRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	if req.DiscLabel == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "disc_label is required")
	}

	ctx := c.Request().Context()
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
				"generateName": "etl-bluray-manual-",
				"namespace":    "pechka",
			},
			"spec": map[string]interface{}{
				"workflowTemplateRef": map[string]interface{}{
					"name": "etl-bluray",
				},
				"entrypoint": "manual",
				"arguments": map[string]interface{}{
					"parameters": []interface{}{
						map[string]interface{}{
							"name":  "disc-label",
							"value": req.DiscLabel,
						},
						map[string]interface{}{
							"name":  "content-title",
							"value": req.ContentTitle,
						},
					},
				},
			},
		},
	}

	created, err := h.dynClient.Resource(workflowGVR).Namespace("pechka").Create(ctx, wf, metav1.CreateOptions{})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	wfName := created.GetName()

	return c.JSON(http.StatusAccepted, IngestResponse{
		Message:      "Ingest workflow successfully submitted",
		WorkflowName: wfName,
	})
}
