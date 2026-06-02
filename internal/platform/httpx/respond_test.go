package httpx

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestErrorResponseShape(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(WithRequestID(req.Context(), "req_test"))
	rec := httptest.NewRecorder()

	Error(rec, req, http.StatusBadRequest, "request.invalid", "请求不正确", nil)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
	var body ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Code != "request.invalid" || body.Message != "请求不正确" || body.RequestID != "req_test" {
		t.Fatalf("unexpected body: %+v", body)
	}
}
