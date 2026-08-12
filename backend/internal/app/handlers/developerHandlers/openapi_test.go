package developerHandlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOpenAPIContainsPathParametersAndResponses(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/openapi.json", nil)
	new(Handler).OpenAPI(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d", recorder.Code)
	}
	var document map[string]interface{}
	if err := json.NewDecoder(recorder.Body).Decode(&document); err != nil {
		t.Fatalf("decode OpenAPI: %v", err)
	}
	paths, ok := document["paths"].(map[string]interface{})
	if !ok {
		t.Fatalf("paths are missing")
	}
	linkPath, ok := paths["/links/{shortCode}"].(map[string]interface{})
	if !ok || linkPath["parameters"] == nil {
		t.Fatalf("shortCode path parameter is missing")
	}
	for _, operationName := range []string{"get", "patch", "delete"} {
		operation, ok := linkPath[operationName].(map[string]interface{})
		if !ok || operation["responses"] == nil {
			t.Fatalf("%s responses are missing", operationName)
		}
	}
}
