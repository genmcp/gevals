package steps

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"
)

func TestSplitPath(t *testing.T) {
	tt := map[string]struct {
		path     string
		expected []pathPart
	}{
		"simple key": {
			path:     "name",
			expected: []pathPart{{key: "name", isIndex: false}},
		},
		"nested keys": {
			path:     "user.name",
			expected: []pathPart{{key: "user", isIndex: false}, {key: "name", isIndex: false}},
		},
		"array index": {
			path:     "items[0]",
			expected: []pathPart{{key: "items", isIndex: false}, {key: "0", isIndex: true}},
		},
		"array index with nested key": {
			path:     "items[0].name",
			expected: []pathPart{{key: "items", isIndex: false}, {key: "0", isIndex: true}, {key: "name", isIndex: false}},
		},
		"deeply nested with array": {
			path:     "data.users[2].email",
			expected: []pathPart{{key: "data", isIndex: false}, {key: "users", isIndex: false}, {key: "2", isIndex: true}, {key: "email", isIndex: false}},
		},
		"numeric string key (not array)": {
			path:     "data.0.field",
			expected: []pathPart{{key: "data", isIndex: false}, {key: "0", isIndex: false}, {key: "field", isIndex: false}},
		},
		"multiple array indices": {
			path:     "matrix[0][1]",
			expected: []pathPart{{key: "matrix", isIndex: false}, {key: "0", isIndex: true}, {key: "1", isIndex: true}},
		},
		"array at root": {
			path:     "[0].name",
			expected: []pathPart{{key: "0", isIndex: true}, {key: "name", isIndex: false}},
		},
	}

	for tn, tc := range tt {
		t.Run(tn, func(t *testing.T) {
			got := splitPath(tc.path)
			assert.Equal(t, tc.expected, got)
		})
	}
}

func TestHttpBody_Validate(t *testing.T) {
	tt := map[string]struct {
		body      *HttpBody
		expectErr bool
	}{
		"valid raw body": {
			body: &HttpBody{
				Raw: ptr.To("hello world"),
			},
			expectErr: false,
		},
		"valid json body": {
			body: &HttpBody{
				JSON: map[string]any{"key": "value"},
			},
			expectErr: false,
		},
		"invalid: both raw and json set": {
			body: &HttpBody{
				Raw:  ptr.To("hello"),
				JSON: map[string]any{"key": "value"},
			},
			expectErr: true,
		},
		"invalid: neither raw nor json set": {
			body:      &HttpBody{},
			expectErr: true,
		},
	}

	for tn, tc := range tt {
		t.Run(tn, func(t *testing.T) {
			err := tc.body.Validate()
			if tc.expectErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestHttpExpect_ValidateResponse(t *testing.T) {
	tt := map[string]struct {
		expect   *HttpExpect
		respBody string
		respCode int
		expected *StepOutput
	}{
		"status matches": {
			expect:   &HttpExpect{Status: 200},
			respCode: 200,
			respBody: "",
			expected: &StepOutput{
				Type:    "http",
				Success: true,
				Message: "response passed all validation",
			},
		},
		"status does not match": {
			expect:   &HttpExpect{Status: 200},
			respCode: 404,
			respBody: "",
			expected: &StepOutput{
				Type:    "http",
				Success: false,
				Error:   "response failed validation check: expected status code 200, got 404",
			},
		},
		"nil expect": {
			expect:   nil,
			respCode: 200,
			respBody: "",
			expected: &StepOutput{
				Type:    "http",
				Success: true,
				Message: "request completed (no expectations defined)",
			},
		},
	}

	for tn, tc := range tt {
		t.Run(tn, func(t *testing.T) {
			resp := &http.Response{
				StatusCode: tc.respCode,
				Body:       io.NopCloser(strings.NewReader(tc.respBody)),
			}
			got := tc.expect.ValidateResponse(resp)
			assert.Equal(t, tc.expected, got)
		})
	}
}

func TestExpectBody_Validate(t *testing.T) {
	tt := map[string]struct {
		expect     *ExpectBody
		body       string
		wantErrors []string
	}{
		"nil expect returns no errors": {
			expect:     nil,
			body:       `{"foo": "bar"}`,
			wantErrors: nil,
		},
		"empty body with no assertions": {
			expect:     &ExpectBody{},
			body:       "",
			wantErrors: nil,
		},
		"match succeeds": {
			expect:     &ExpectBody{Match: ptr.To(`"status":\s*"ok"`)},
			body:       `{"status": "ok"}`,
			wantErrors: nil,
		},
		"match fails": {
			expect:     &ExpectBody{Match: ptr.To(`"status":\s*"ok"`)},
			body:       `{"status": "error"}`,
			wantErrors: []string{`body did not match pattern "\"status\":\\s*\"ok\""`},
		},
		"match on empty body fails": {
			expect:     &ExpectBody{Match: ptr.To(`something`)},
			body:       "",
			wantErrors: []string{`body did not match pattern "something"`},
		},
		"field equals succeeds": {
			expect: &ExpectBody{
				Fields: []FieldAssertion{{Path: "name", Equals: "test"}},
			},
			body:       `{"name": "test"}`,
			wantErrors: nil,
		},
		"field equals fails": {
			expect: &ExpectBody{
				Fields: []FieldAssertion{{Path: "name", Equals: "test"}},
			},
			body:       `{"name": "other"}`,
			wantErrors: []string{`field "name": expected test, got other`},
		},
		"nested field succeeds": {
			expect: &ExpectBody{
				Fields: []FieldAssertion{{Path: "user.name", Equals: "alice"}},
			},
			body:       `{"user": {"name": "alice"}}`,
			wantErrors: nil,
		},
		"array index field succeeds": {
			expect: &ExpectBody{
				Fields: []FieldAssertion{{Path: "items[0].id", Equals: float64(1)}},
			},
			body:       `{"items": [{"id": 1}, {"id": 2}]}`,
			wantErrors: nil,
		},
		"numeric string key in object succeeds": {
			expect: &ExpectBody{
				Fields: []FieldAssertion{{Path: "data.0.value", Equals: "first"}},
			},
			body:       `{"data": {"0": {"value": "first"}, "1": {"value": "second"}}}`,
			wantErrors: nil,
		},
		"array index second element": {
			expect: &ExpectBody{
				Fields: []FieldAssertion{{Path: "items[1].id", Equals: float64(2)}},
			},
			body:       `{"items": [{"id": 1}, {"id": 2}, {"id": 3}]}`,
			wantErrors: nil,
		},
		"nested array index": {
			expect: &ExpectBody{
				Fields: []FieldAssertion{{Path: "data.users[1].name", Equals: "bob"}},
			},
			body:       `{"data": {"users": [{"name": "alice"}, {"name": "bob"}]}}`,
			wantErrors: nil,
		},
		"array index out of bounds": {
			expect: &ExpectBody{
				Fields: []FieldAssertion{{Path: "items[5].id", Equals: float64(1)}},
			},
			body:       `{"items": [{"id": 1}, {"id": 2}]}`,
			wantErrors: []string{`field "items[5].id" does not exist`},
		},
		"array index on non-array fails": {
			expect: &ExpectBody{
				Fields: []FieldAssertion{{Path: "name[0]", Equals: "a"}},
			},
			body:       `{"name": "alice"}`,
			wantErrors: []string{`field "name[0]" does not exist`},
		},
		"multi-dimensional array": {
			expect: &ExpectBody{
				Fields: []FieldAssertion{{Path: "matrix[1][0]", Equals: float64(3)}},
			},
			body:       `{"matrix": [[1, 2], [3, 4]]}`,
			wantErrors: nil,
		},
		"root level array": {
			expect: &ExpectBody{
				Fields: []FieldAssertion{{Path: "[0].id", Equals: float64(1)}},
			},
			body:       `[{"id": 1}, {"id": 2}]`,
			wantErrors: nil,
		},
		"root level array second element": {
			expect: &ExpectBody{
				Fields: []FieldAssertion{{Path: "[1].name", Equals: "bob"}},
			},
			body:       `[{"name": "alice"}, {"name": "bob"}]`,
			wantErrors: nil,
		},
		"root level array out of bounds": {
			expect: &ExpectBody{
				Fields: []FieldAssertion{{Path: "[5].id", Equals: float64(1)}},
			},
			body:       `[{"id": 1}, {"id": 2}]`,
			wantErrors: []string{`field "[5].id" does not exist`},
		},
		"field type check succeeds": {
			expect: &ExpectBody{
				Fields: []FieldAssertion{{Path: "count", Type: "number"}},
			},
			body:       `{"count": 42}`,
			wantErrors: nil,
		},
		"field type check fails": {
			expect: &ExpectBody{
				Fields: []FieldAssertion{{Path: "count", Type: "string"}},
			},
			body:       `{"count": 42}`,
			wantErrors: []string{`field "count": expected type string, got number`},
		},
		"field exists check succeeds": {
			expect: &ExpectBody{
				Fields: []FieldAssertion{{Path: "id", Exists: ptr.To(true)}},
			},
			body:       `{"id": 123}`,
			wantErrors: nil,
		},
		"field exists check fails": {
			expect: &ExpectBody{
				Fields: []FieldAssertion{{Path: "id", Exists: ptr.To(true)}},
			},
			body:       `{"name": "test"}`,
			wantErrors: []string{`field "id" does not exist`},
		},
		"field not exists check succeeds": {
			expect: &ExpectBody{
				Fields: []FieldAssertion{{Path: "deleted", Exists: ptr.To(false)}},
			},
			body:       `{"id": 123}`,
			wantErrors: nil,
		},
		"field not exists check fails": {
			expect: &ExpectBody{
				Fields: []FieldAssertion{{Path: "deleted", Exists: ptr.To(false)}},
			},
			body:       `{"deleted": true}`,
			wantErrors: []string{`field "deleted" exists but should not`},
		},
		"field match regex succeeds": {
			expect: &ExpectBody{
				Fields: []FieldAssertion{{Path: "email", Match: ptr.To(`^[a-z]+@example\.com$`)}},
			},
			body:       `{"email": "test@example.com"}`,
			wantErrors: nil,
		},
		"field match regex fails": {
			expect: &ExpectBody{
				Fields: []FieldAssertion{{Path: "email", Match: ptr.To(`^[a-z]+@example\.com$`)}},
			},
			body:       `{"email": "invalid"}`,
			wantErrors: []string{`field "email": value "invalid" did not match pattern "^[a-z]+@example\\.com$"`},
		},
		"empty body with field assertions fails": {
			expect: &ExpectBody{
				Fields: []FieldAssertion{{Path: "id", Type: "number"}},
			},
			body:       "",
			wantErrors: []string{"expected JSON body but got empty response"},
		},
		"invalid json fails": {
			expect: &ExpectBody{
				Fields: []FieldAssertion{{Path: "id", Type: "number"}},
			},
			body:       "not json",
			wantErrors: []string{"failed to parse response body as JSON: invalid character 'o' in literal null (expecting 'u')"},
		},
		"multiple assertions all pass": {
			expect: &ExpectBody{
				Match: ptr.To(`"status"`),
				Fields: []FieldAssertion{
					{Path: "status", Equals: "ok"},
					{Path: "count", Type: "number"},
				},
			},
			body:       `{"status": "ok", "count": 5}`,
			wantErrors: nil,
		},
		"multiple assertions some fail": {
			expect: &ExpectBody{
				Fields: []FieldAssertion{
					{Path: "status", Equals: "ok"},
					{Path: "count", Type: "string"},
				},
			},
			body:       `{"status": "error", "count": 5}`,
			wantErrors: []string{`field "status": expected ok, got error`, `field "count": expected type string, got number`},
		},
		"int equals float64 (YAML int vs JSON float64)": {
			expect: &ExpectBody{
				Fields: []FieldAssertion{{Path: "total", Equals: int(7500)}},
			},
			body:       `{"total": 7500.00}`,
			wantErrors: nil,
		},
		"float64 equals int (JSON float64 vs YAML int)": {
			expect: &ExpectBody{
				Fields: []FieldAssertion{{Path: "total", Equals: float64(7500)}},
			},
			body:       `{"total": 7500}`,
			wantErrors: nil,
		},
		"int64 equals float64": {
			expect: &ExpectBody{
				Fields: []FieldAssertion{{Path: "total", Equals: int64(7500)}},
			},
			body:       `{"total": 7500.00}`,
			wantErrors: nil,
		},
	}

	for tn, tc := range tt {
		t.Run(tn, func(t *testing.T) {
			got := tc.expect.Validate([]byte(tc.body))
			assert.Equal(t, tc.wantErrors, got)
		})
	}
}

func TestHttpBody_Content(t *testing.T) {
	tt := map[string]struct {
		body              *HttpBody
		expectErr         bool
		expectEmpty       bool
		expectContentType string
	}{
		"nil body returns empty reader": {
			body:              nil,
			expectErr:         false,
			expectEmpty:       true,
			expectContentType: "",
		},
		"raw body returns reader with content": {
			body: &HttpBody{
				Raw: ptr.To("test content"),
			},
			expectErr:         false,
			expectEmpty:       false,
			expectContentType: "",
		},
		"json body returns reader with marshaled content and content type": {
			body: &HttpBody{
				JSON: map[string]any{"foo": "bar"},
			},
			expectErr:         false,
			expectEmpty:       false,
			expectContentType: "application/json",
		},
	}

	for tn, tc := range tt {
		t.Run(tn, func(t *testing.T) {
			content, err := tc.body.Content()
			if tc.expectErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, content)
			assert.NotNil(t, content.Reader)
			assert.Equal(t, tc.expectContentType, content.ContentType)
		})
	}
}

func TestHttpStep_Execute(t *testing.T) {
	tt := map[string]struct {
		handler   http.HandlerFunc
		config    *HttpStepConfig
		input     *StepInput
		expected  *StepOutput
		expectErr bool
	}{
		"GET request returns expected status": {
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			config: &HttpStepConfig{
				Method: "GET",
				Body:   &HttpBody{Raw: ptr.To("")},
				Expect: &HttpExpect{Status: 200},
			},
			input: &StepInput{Env: map[string]string{}},
			expected: &StepOutput{
				Type:    "http",
				Success: true,
				Message: "response passed all validation",
			},
			expectErr: false,
		},
		"POST request with JSON body sets Content-Type header": {
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}
				if r.Header.Get("Content-Type") != "application/json" {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				w.WriteHeader(http.StatusCreated)
			},
			config: &HttpStepConfig{
				Method: "POST",
				Body:   &HttpBody{JSON: map[string]any{"name": "test"}},
				Expect: &HttpExpect{Status: 201},
			},
			input: &StepInput{Env: map[string]string{}},
			expected: &StepOutput{
				Type:    "http",
				Success: true,
				Message: "response passed all validation",
			},
			expectErr: false,
		},
		"request returns unexpected status": {
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
			config: &HttpStepConfig{
				Method: "GET",
				Body:   &HttpBody{Raw: ptr.To("")},
				Expect: &HttpExpect{Status: 200},
			},
			input: &StepInput{Env: map[string]string{}},
			expected: &StepOutput{
				Type:    "http",
				Success: false,
				Error:   "response failed validation check: expected status code 200, got 404",
			},
			expectErr: false,
		},
	}

	for tn, tc := range tt {
		t.Run(tn, func(t *testing.T) {
			server := httptest.NewServer(tc.handler)
			defer server.Close()

			tc.config.URL = server.URL

			step, err := NewHttpStep(tc.config)
			require.NoError(t, err)

			got, err := step.Execute(context.Background(), tc.input)
			if tc.expectErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.expected, got)
		})
	}
}
