package testutil

import (
	"encoding/json"
	"errors"
	"net/http/httptest"

	"investment_monitoring/pkg/errs"
)

type TestingT interface {
	Helper()
	Fatalf(format string, args ...any)
}

type ResponseBody struct {
	ResponseMessage  string            `json:"responseMessage"`
	Data             any               `json:"data"`
	ValidationErrors map[string]string `json:"validationErrors"`
}

func AssertAppError(t TestingT, err error, wantMessage string) {
	t.Helper()

	if err == nil {
		t.Fatalf("err = nil, want %s", wantMessage)
	}
	var appErr *errs.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("err = %T %v, want AppError", err, err)
	}
	if appErr.Message != wantMessage {
		t.Fatalf("message = %s, want %s", appErr.Message, wantMessage)
	}
}

func AssertResponse(t TestingT, rr *httptest.ResponseRecorder, wantStatus int, wantMessage string) {
	t.Helper()

	if rr.Code != wantStatus {
		t.Fatalf("status = %d, want %d; body=%s", rr.Code, wantStatus, rr.Body.String())
	}
	var body ResponseBody
	DecodeBody(t, rr, &body)
	if body.ResponseMessage != wantMessage {
		t.Fatalf("response message = %s, want %s", body.ResponseMessage, wantMessage)
	}
}

func DecodeBody(t TestingT, rr *httptest.ResponseRecorder, dest any) {
	t.Helper()

	if err := json.Unmarshal(rr.Body.Bytes(), dest); err != nil {
		t.Fatalf("decode body: %v; body=%s", err, rr.Body.String())
	}
}
