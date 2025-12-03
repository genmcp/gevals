package steps

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/genmcp/gen-mcp/pkg/template"
)

type HttpStepConfig struct {
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    *HttpBody         `json:"body,omitempty"`
	Expect  *HttpExpect       `json:"expect,omitempty"`
	Timeout string            `json:"timeout,omitempty"`
}

type HttpBody struct {
	Raw  *string        `json:"raw,omitempty"`
	JSON map[string]any `json:"json,omitempty"` // TODO: find a way to handle possibly templated values in the body
}

type HttpExpect struct {
	Status int `json:"status,omitempty"`
}

type HttpStep struct {
	URL     *template.TemplateBuilder
	Method  *template.TemplateBuilder
	Headers map[string]*template.TemplateBuilder
	Body    *HttpBody
	Expect  *HttpExpect
	Timeout time.Duration
}

var _ StepRunner = &HttpStep{}

func ParseHttpStep(raw json.RawMessage) (StepRunner, error) {
	cfg := &HttpStepConfig{}

	err := json.Unmarshal(raw, cfg)
	if err != nil {
		return nil, err
	}

	return NewHttpStep(cfg)
}

func NewHttpStep(cfg *HttpStepConfig) (*HttpStep, error) {
	var err error
	step := &HttpStep{}

	url, err := template.ParseTemplate(cfg.URL, template.TemplateParserOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to parse url: %w", err)
	}

	step.URL, err = template.NewTemplateBuilder(url, false)
	if err != nil {
		return nil, fmt.Errorf("failed to create builder for url: %w", err)
	}

	method, err := template.ParseTemplate(cfg.Method, template.TemplateParserOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to parse method: %w", err)
	}

	step.Method, err = template.NewTemplateBuilder(method, false)
	if err != nil {
		return nil, fmt.Errorf("failed to create builder for method: %w", err)
	}

	step.Headers = make(map[string]*template.TemplateBuilder, len(cfg.Headers))
	for k, v := range cfg.Headers {
		h, err := template.ParseTemplate(v, template.TemplateParserOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to parse header: %w", err)
		}

		step.Headers[k], err = template.NewTemplateBuilder(h, false)
		if err != nil {
			return nil, fmt.Errorf("failed to create builder for header: %w", err)
		}
	}

	step.Body = cfg.Body
	if err := step.Body.Validate(); err != nil {
		return nil, fmt.Errorf("invalid body for http step: %w", err)
	}

	step.Expect = cfg.Expect

	if cfg.Timeout != "" {
		timeout, err := time.ParseDuration(cfg.Timeout)
		if err != nil {
			return nil, fmt.Errorf("failed to parse timeout: %w", err)
		}
		step.Timeout = timeout
	} else {
		step.Timeout = DefaultTimout
	}

	return step, nil
}

func (s *HttpStep) Execute(ctx context.Context, input *StepInput) (*StepOutput, error) {
	for k, v := range input.Env {
		err := os.Setenv(k, v)
		if err != nil {
			return nil, fmt.Errorf("failed to set env var '%s' to value '%s': %w", k, v, err)
		}
	}
	defer func() {
		for k := range input.Env {
			_ = os.Unsetenv(k)
		}
	}()

	method, err := s.Method.GetResult()
	if err != nil {
		return nil, fmt.Errorf("failed to build method from template: %w", err)
	}

	url, err := s.URL.GetResult()
	if err != nil {
		return nil, fmt.Errorf("failed to build url from template: %w", err)
	}

	body, err := s.Body.Reader()
	if err != nil {
		return nil, fmt.Errorf("failed to create reader for request body: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, s.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, method.(string), url.(string), body)

	client := http.DefaultClient

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make http request: %w", err)
	}

	return s.Expect.ValidateResponse(resp), nil
}

func (b *HttpBody) Reader() (io.Reader, error) {
	if b == nil {
		return bytes.NewReader(nil), nil
	}

	if b.Raw != nil {
		return strings.NewReader(*b.Raw), nil
	}
	if b.JSON != nil {
		b, err := json.Marshal(b.JSON)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal body.json to json: %s", err)
		}

		return bytes.NewReader(b), nil
	}

	return nil, fmt.Errorf("no valid body set")
}

func (b *HttpBody) Validate() error {
	numDefined := 0
	if b.Raw != nil {
		numDefined++
	}
	if b.JSON != nil {
		numDefined++
	}

	if numDefined != 1 {
		return fmt.Errorf("exactly one key must be defined on body")
	}

	return nil
}

func (e *HttpExpect) ValidateResponse(resp *http.Response) *StepOutput {
	success := e.Status == resp.StatusCode
	out := &StepOutput{
		Success: success,
	}

	if success {
		out.Message = "response passed all validation"
	}

	if !success {
		out.Error = fmt.Sprintf("response failed validation check: expected status code %d, got %d", e.Status, resp.StatusCode)
	}

	return out
}
