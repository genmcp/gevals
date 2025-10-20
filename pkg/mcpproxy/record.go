package mcpproxy

import (
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type Recorder interface {
	RecordToolCall(req *mcp.CallToolRequest, res *mcp.CallToolResult, err error, start time.Time)
	RecordResourceRead(req *mcp.ReadResourceRequest, res *mcp.ReadResourceResult, err error, start time.Time)
	RecordPromptGet(req *mcp.GetPromptRequest, res *mcp.GetPromptResult, err error, start time.Time)
	GetHistory() CallHistory
}

// CallRecord is the base for all MCP interaction types
type CallRecord struct {
	ServerName string    `json:"serverName"`
	Timestamp  time.Time `json:"timestamp"`
	Success    bool      `json:"success"`
	Error      string    `json:"error,omitempty"`
}

// ToolCall records a tool invocation
type ToolCall struct {
	CallRecord
	ToolName string               `json:"name"` // this is copied to the top level struct for convenience
	Request  *mcp.CallToolRequest `json:"request,omitempty"`
	Result   *mcp.CallToolResult  `json:"result,omitempty"`
}

// ResourceRead records a resource read
type ResourceRead struct {
	CallRecord
	URI     string                   `json:"uri"` // this is copied to the top level struct for convenience
	Request *mcp.ReadResourceRequest `json:"request"`
	Result  *mcp.ReadResourceResult  `json:"result"`
}

// PromptGet records a prompt get
type PromptGet struct {
	CallRecord
	Name    string                `json:"name"` // this is copies to the top level struct for convenience
	Request *mcp.GetPromptRequest `json:"request"`
	Result  *mcp.GetPromptResult  `json:"result"`
}

// CallHistory contains a complete call history for a server
type CallHistory struct {
	ToolCalls     []*ToolCall
	ResourceReads []*ResourceRead
	PromptGets    []*PromptGet
}

type recorder struct {
	serverName string

	mu      sync.RWMutex
	history *CallHistory
}

var _ Recorder = &recorder{}

func NewRecorder(serverName string) Recorder {
	return &recorder{
		serverName: serverName,
		history: &CallHistory{
			ToolCalls:     make([]*ToolCall, 0),
			ResourceReads: make([]*ResourceRead, 0),
			PromptGets:    make([]*PromptGet, 0),
		},
	}
}

func (r *recorder) RecordToolCall(req *mcp.CallToolRequest, res *mcp.CallToolResult, err error, start time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.history.ToolCalls = append(r.history.ToolCalls, &ToolCall{
		CallRecord: CallRecord{
			ServerName: r.serverName,
			Timestamp:  start,
			Success:    err == nil,
			Error:      errorToString(err),
		},
		ToolName: req.Params.Name,
		Request:  req,
		Result:   res,
	})
}

func (r *recorder) RecordResourceRead(req *mcp.ReadResourceRequest, res *mcp.ReadResourceResult, err error, start time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.history.ResourceReads = append(r.history.ResourceReads, &ResourceRead{
		CallRecord: CallRecord{
			ServerName: r.serverName,
			Timestamp:  start,
			Success:    err == nil,
			Error:      errorToString(err),
		},
		URI:     req.Params.URI,
		Request: req,
		Result:  res,
	})
}

func (r *recorder) RecordPromptGet(req *mcp.GetPromptRequest, res *mcp.GetPromptResult, err error, start time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.history.PromptGets = append(r.history.PromptGets, &PromptGet{
		CallRecord: CallRecord{
			ServerName: r.serverName,
			Timestamp:  start,
			Success:    err == nil,
			Error:      errorToString(err),
		},
		Name:    req.Params.Name,
		Request: req,
		Result:  res,
	})
}

func (r *recorder) GetHistory() CallHistory {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return *r.history
}

func errorToString(err error) string {
	if err == nil {
		return ""
	}

	return err.Error()
}
