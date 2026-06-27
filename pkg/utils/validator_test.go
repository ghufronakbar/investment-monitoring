package utils

import (
	"strings"
	"testing"
)

type validationTestInput struct {
	Name  string                 `json:"name" validate:"required,min=2"`
	Email string                 `json:"email" validate:"required,email"`
	Items []validationTestItem   `json:"items" validate:"required,min=1"`
	Meta  validationTestMetadata `json:"meta"`
}

type validationTestItem struct {
	Title string `json:"title" validate:"required"`
}

type validationTestMetadata struct {
	Note string `json:"note"`
}

func TestValidateStructSuccess(t *testing.T) {
	var dst validationTestInput
	err := ValidateStruct(strings.NewReader(`{
		"name": "Jane",
		"email": "jane@example.com",
		"items": [{"title": "Task"}],
		"meta": {"note": "ok"}
	}`), &dst)
	if err != nil {
		t.Fatalf("ValidateStruct returned error: %+v", err)
	}
	if dst.Name != "Jane" || dst.Items[0].Title != "Task" {
		t.Fatalf("dst = %+v, want decoded payload", dst)
	}
}

func TestValidateStructCollectsUnknownNestedAndValidationErrors(t *testing.T) {
	var dst validationTestInput
	err := ValidateStruct(strings.NewReader(`{
		"name": "J",
		"email": "not-email",
		"unknown": true,
		"items": [{"title": "", "extra": "x"}],
		"meta": {"note": "ok", "extra": "x"}
	}`), &dst)
	if err == nil {
		t.Fatal("expected validation error")
	}

	want := map[string]string{
		"unknown":        "unknown field",
		"items[0].extra": "unknown field",
		"meta.extra":     "unknown field",
		"name":           "must be at least 2",
		"email":          "must be a valid email",
	}
	for field, message := range want {
		if err.ValidationErrors[field] != message {
			t.Fatalf("validation error %s = %q, want %q; all errors: %+v", field, err.ValidationErrors[field], message, err.ValidationErrors)
		}
	}
}

func TestValidateStructRejectsInvalidBodyShapes(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{name: "empty", body: ``, want: "request body is empty"},
		{name: "invalid json", body: `{`, want: "invalid JSON format"},
		{name: "multiple values", body: `{} {}`, want: "invalid JSON: multiple JSON values in request body"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var dst validationTestInput
			err := ValidateStruct(strings.NewReader(tt.body), &dst)
			if err == nil {
				t.Fatal("expected validation error")
			}
			if err.ValidationErrors["_error"] != tt.want {
				t.Fatalf("_error = %q, want %q", err.ValidationErrors["_error"], tt.want)
			}
		})
	}
}

func TestValidateStructRejectsInvalidDestination(t *testing.T) {
	if err := ValidateStruct(strings.NewReader(`{}`), nil); err == nil || err.ValidationErrors["_error"] != "destination is nil" {
		t.Fatalf("nil destination err = %+v, want destination is nil", err)
	}

	var dst validationTestInput
	if err := ValidateStruct(strings.NewReader(`{}`), dst); err == nil || err.ValidationErrors["_error"] != "destination must be pointer to struct" {
		t.Fatalf("non pointer destination err = %+v, want destination must be pointer to struct", err)
	}
}
