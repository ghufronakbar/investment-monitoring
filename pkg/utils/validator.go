// pkg/utils/validator.go
package utils

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"

	"investment_monitoring/pkg/errs"
)

// validator singleton
var v = validator.New()

// ValidateStruct:
// - baca raw body (mentah)
// - parse JSON sekali untuk cek syntax + memastikan hanya 1 JSON value
// - scan semua unknown fields (berdasarkan json tag struct) TANPA berhenti di error pertama
// - unmarshal ke struct (type mismatch: stdlib hanya kasih 1 error pertama)
// - validate struct pakai tags validate:"..."
// - kalau ada error → return errs.NewValidationFailed(map[string]string)
func ValidateStruct(body io.Reader, dst any) *errs.AppError {
	// dst WAJIB pointer ke struct
	if dst == nil {
		return errs.NewValidationFailed(map[string]string{
			"_error": "destination is nil",
		})
	}

	t := reflect.TypeOf(dst)
	if t.Kind() != reflect.Ptr || t.Elem().Kind() != reflect.Struct {
		return errs.NewValidationFailed(map[string]string{
			"_error": "destination must be pointer to struct",
		})
	}
	rootType := t.Elem()

	// 0) Baca body jadi bytes agar bisa di-parse beberapa kali tanpa EOF
	rawBody, err := io.ReadAll(body)
	if err != nil {
		return errs.NewValidationFailed(map[string]string{
			"_error": "failed to read request body",
		})
	}
	if len(bytes.TrimSpace(rawBody)) == 0 {
		return errs.NewValidationFailed(map[string]string{
			"_error": "request body is empty",
		})
	}

	// 1) Decode ke `any` untuk:
	//    - detect invalid JSON syntax
	//    - detect multiple JSON values (contoh: {...}{...})
	//    - bahan scan unknown fields lengkap
	var raw any
	dec := json.NewDecoder(bytes.NewReader(rawBody))
	dec.UseNumber() // biar angka tidak langsung jadi float64, lebih aman untuk inspect

	if err := dec.Decode(&raw); err != nil {
		// invalid JSON format / syntax
		return errs.NewValidationFailed(parseDecodeError(err, rootType))
	}

	// Pastikan tidak ada JSON value kedua
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return errs.NewValidationFailed(map[string]string{
			"_error": "invalid JSON: multiple JSON values in request body",
		})
	}

	// 2) Scan semua unknown fields berdasarkan schema struct (json tag)
	unknowns := make(map[string]string)
	collectUnknownFields(raw, rootType, "", unknowns)

	// 3) Unmarshal ke struct (tanpa DisallowUnknownFields),
	//    supaya kita tidak stop di error unknown field pertama.
	//    Type mismatch tetap bisa terjadi.
	typeErrs := make(map[string]string)
	if err := json.Unmarshal(rawBody, dst); err != nil {
		// Ambil error type mismatch / dll jadi map
		for k, v := range parseDecodeError(err, rootType) {
			typeErrs[k] = v
		}
		// NOTE: json.Unmarshal hanya mengembalikan 1 error pertama untuk type mismatch.
		// Tapi kita tetap lanjut ke validator agar user dapat banyak error lain sekaligus.
	}

	// 4) Validate struct (bisa menghasilkan banyak error sekaligus)
	valErrs := make(map[string]string)
	if err := v.Struct(dst); err != nil {
		if ve, ok := err.(validator.ValidationErrors); ok {
			for _, fe := range ve {
				jsonPath := namespaceToJSONPath(rootType, fe.Namespace())
				if jsonPath == "" {
					jsonPath = lowerFirst(fe.Field())
				}
				valErrs[jsonPath] = messageForValidationError(fe)
			}
		} else {
			valErrs["_error"] = "validation failed"
		}
	}

	// 5) Gabungkan semua error: unknown + type mismatch + validator
	merged := mergeErrors(unknowns, typeErrs, valErrs)
	if len(merged) > 0 {
		return errs.NewValidationFailed(merged)
	}

	return nil
}

// mergeErrors menggabungkan beberapa map error tanpa overwrite yang sudah ada.
// Prioritas (yang ditaruh duluan tidak ditimpa):
// unknown fields -> type mismatch -> validation errors
func mergeErrors(maps ...map[string]string) map[string]string {
	out := map[string]string{}
	for _, m := range maps {
		for k, v := range m {
			if _, exists := out[k]; exists {
				continue
			}
			out[k] = v
		}
	}
	return out
}

// =========================
// Decode error → map field->message
// =========================
func parseDecodeError(err error, rootType reflect.Type) map[string]string {
	// Body kosong (biasanya muncul kalau decode streaming)
	if errors.Is(err, io.EOF) {
		return map[string]string{"_error": "request body is empty"}
	}

	// JSON syntax rusak
	var se *json.SyntaxError
	if errors.As(err, &se) {
		return map[string]string{"_error": "invalid JSON syntax"}
	}

	// Tipe salah (misalnya number ke string)
	var ute *json.UnmarshalTypeError
	if errors.As(err, &ute) {
		key := "_error"
		if ute.Field != "" {
			// ute.Field biasanya path Go field (kadang "BusinessKnowledge.PrimaryLogoUrl")
			key = goPathToJSONPath(rootType, ute.Field)
			if key == "" {
				key = ute.Field
			}
		}
		return map[string]string{
			key: fmt.Sprintf("must be %s", ute.Type.String()),
		}
	}

	// Unknown field dari DisallowUnknownFields (kalau suatu saat kamu pakai lagi)
	msg := err.Error()
	if strings.HasPrefix(msg, `json: unknown field "`) {
		f := strings.TrimPrefix(msg, `json: unknown field "`)
		f = strings.TrimSuffix(f, `"`)
		return map[string]string{
			f: "unknown field",
		}
	}

	return map[string]string{"_error": "invalid JSON format"}
}

// =========================
// Unknown field scanner
// =========================

// collectUnknownFields: scan JSON object dan bandingkan dengan schema struct.
// Output key berupa json path: "businessKnowledge.primaryLogo"
func collectUnknownFields(raw any, structType reflect.Type, path string, out map[string]string) {
	// unwrap pointer
	for structType.Kind() == reflect.Ptr {
		structType = structType.Elem()
	}
	if structType.Kind() != reflect.Struct {
		return
	}

	obj, ok := raw.(map[string]any)
	if !ok {
		// kalau JSON bukan object, biarkan type mismatch ditangani parseDecodeError
		return
	}

	allowed := schemaFields(structType) // jsonName -> fieldType

	for k, v := range obj {
		fieldType, exists := allowed[k]
		full := joinPath(path, k)

		if !exists {
			out[full] = "unknown field"
			continue
		}

		// recursive check hanya kalau target field itu struct / slice of struct
		inspectValueAgainstField(v, fieldType, full, out)
	}
}

func inspectValueAgainstField(val any, fieldType reflect.Type, path string, out map[string]string) {
	// unwrap pointer
	for fieldType.Kind() == reflect.Ptr {
		fieldType = fieldType.Elem()
	}

	switch fieldType.Kind() {
	case reflect.Struct:
		// treat time.Time as scalar
		if isTimeType(fieldType) {
			return
		}
		collectUnknownFields(val, fieldType, path, out)

	case reflect.Slice, reflect.Array:
		elem := fieldType.Elem()
		// unwrap pointer elem
		for elem.Kind() == reflect.Ptr {
			elem = elem.Elem()
		}

		arr, ok := val.([]any)
		if !ok {
			return
		}

		// kalau elem struct (bukan time), scan per item
		if elem.Kind() == reflect.Struct && !isTimeType(elem) {
			for i, item := range arr {
				itemPath := fmt.Sprintf("%s[%d]", path, i)
				collectUnknownFields(item, elem, itemPath, out)
			}
		}

	default:
		// primitives/map/etc: tidak perlu scan unknown nested
		return
	}
}

func schemaFields(t reflect.Type) map[string]reflect.Type {
	out := map[string]reflect.Type{}
	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)

		// skip unexported
		if sf.PkgPath != "" {
			continue
		}

		name := jsonNameOfField(sf)
		if name == "-" {
			continue
		}

		out[name] = sf.Type
	}
	return out
}

func joinPath(prefix, key string) string {
	if prefix == "" {
		return key
	}
	return prefix + "." + key
}

func isTimeType(t reflect.Type) bool {
	return t.PkgPath() == "time" && t.Name() == "Time"
}

// =========================
// Validation error message
// =========================
func messageForValidationError(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "is required"
	case "email":
		return "must be a valid email"
	case "url":
		return "must be a valid URL"
	case "min":
		return fmt.Sprintf("must be at least %s", fe.Param())
	case "max":
		return fmt.Sprintf("must be at most %s", fe.Param())
	case "len":
		return fmt.Sprintf("must have length %s", fe.Param())
	case "gte":
		return fmt.Sprintf("must be >= %s", fe.Param())
	case "lte":
		return fmt.Sprintf("must be <= %s", fe.Param())
	default:
		return "is invalid"
	}
}

// =========================
// Convert namespace (Go field path) -> json tag path
// =========================

// namespaceToJSONPath converts validator namespace into JSON tag path.
// rootType: struct tipe root (bukan pointer)
// ns example: "BusinessSetupInput.BusinessKnowledge.PrimaryLogoUrl"
func namespaceToJSONPath(rootType reflect.Type, ns string) string {
	parts := strings.Split(ns, ".")
	if len(parts) == 0 {
		return ""
	}

	// skip root name (parts[0])
	parts = parts[1:]
	return goPartsToJSONPath(rootType, parts)
}

// goPathToJSONPath converts UnmarshalTypeError.Field path into json path.
// ute.Field kadang "BusinessKnowledge.PrimaryLogoUrl" atau "PrimaryLogoUrl".
func goPathToJSONPath(rootType reflect.Type, goFieldPath string) string {
	parts := strings.Split(goFieldPath, ".")
	if len(parts) == 0 {
		return ""
	}
	return goPartsToJSONPath(rootType, parts)
}

func goPartsToJSONPath(rootType reflect.Type, goFieldParts []string) string {
	cur := rootType
	var jsonParts []string

	for _, goName := range goFieldParts {
		// unwrap pointer/slice/array
		for cur.Kind() == reflect.Ptr || cur.Kind() == reflect.Slice || cur.Kind() == reflect.Array {
			cur = cur.Elem()
		}
		if cur.Kind() != reflect.Struct {
			break
		}

		sf, ok := cur.FieldByName(goName)
		if !ok {
			return ""
		}

		jsonName := jsonNameOfField(sf)
		jsonParts = append(jsonParts, jsonName)

		cur = sf.Type
	}

	return strings.Join(jsonParts, ".")
}

func jsonNameOfField(sf reflect.StructField) string {
	tag := sf.Tag.Get("json")
	if tag == "" {
		return lowerFirst(sf.Name)
	}
	name := strings.Split(tag, ",")[0]
	if name == "" {
		return lowerFirst(sf.Name)
	}
	return name
}

func lowerFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}
