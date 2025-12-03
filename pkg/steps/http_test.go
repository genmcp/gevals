package steps

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"
)

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
		resp     *http.Response
		expected *StepOutput
	}{
		"status matches": {
			expect: &HttpExpect{Status: 200},
			resp:   &http.Response{StatusCode: 200},
			expected: &StepOutput{
				Success: true,
				Message: "response passed all validation",
			},
		},
		"status does not match": {
			expect: &HttpExpect{Status: 200},
			resp:   &http.Response{StatusCode: 404},
			expected: &StepOutput{
				Success: false,
				Error:   "response failed validation check: expected status code 200, got 404",
			},
		},
	}

	for tn, tc := range tt {
		t.Run(tn, func(t *testing.T) {
			got := tc.expect.ValidateResponse(tc.resp)
			assert.Equal(t, tc.expected, got)
		})
	}
}

func TestHttpBody_Reader(t *testing.T) {
	tt := map[string]struct {
		body        *HttpBody
		expectErr   bool
		expectEmpty bool
	}{
		"nil body returns empty reader": {
			body:        nil,
			expectErr:   false,
			expectEmpty: true,
		},
		"raw body returns reader with content": {
			body: &HttpBody{
				Raw: ptr.To("test content"),
			},
			expectErr:   false,
			expectEmpty: false,
		},
		"json body returns reader with marshaled content": {
			body: &HttpBody{
				JSON: map[string]any{"foo": "bar"},
			},
			expectErr:   false,
			expectEmpty: false,
		},
	}

	for tn, tc := range tt {
		t.Run(tn, func(t *testing.T) {
			reader, err := tc.body.Reader()
			if tc.expectErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, reader)
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
				Success: true,
				Message: "response passed all validation",
			},
			expectErr: false,
		},
		"POST request with JSON body": {
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					w.WriteHeader(http.StatusMethodNotAllowed)
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
