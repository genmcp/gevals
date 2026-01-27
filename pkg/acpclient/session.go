package acpclient

import (
	"sync"

	"github.com/coder/acp-go-sdk"
	"github.com/mcpchecker/mcpchecker/pkg/mcpproxy"
)

type session struct {
	mu               sync.Mutex
	updates          []acp.SessionUpdate // track all the updates in a json serializable way for future analysis
	toolCallStatuses map[acp.ToolCallId]*acp.SessionToolCallUpdate
	mcpServers       mcpproxy.ServerManager
}

func NewSession(mcpServers mcpproxy.ServerManager) *session {
	return &session{
		updates:          make([]acp.SessionUpdate, 0),
		toolCallStatuses: make(map[acp.ToolCallId]*acp.SessionToolCallUpdate),
		mcpServers:       mcpServers,
	}
}

func (s *session) IsAllowedToolCall(call acp.RequestPermissionToolCall) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Update tool call statuses within the same critical section to ensure
	// atomicity between the update and subsequent read.
	s.toolCallStatusUpdateLocked(&acp.SessionToolCallUpdate{
		Meta:       call.Meta,
		Content:    call.Content,
		Kind:       call.Kind,
		Locations:  call.Locations,
		RawInput:   call.RawInput,
		RawOutput:  call.RawOutput,
		Status:     call.Status,
		Title:      call.Title,
		ToolCallId: call.ToolCallId,
	})

	var title string
	if call.Title != nil {
		title = *call.Title
	} else {
		// look up the original update with the tool call id
		curr, ok := s.toolCallStatuses[call.ToolCallId]
		if !ok {
			return false
		}

		if curr.Title == nil {
			return false
		}

		title = *curr.Title
	}

	for _, srv := range s.mcpServers.GetMcpServers() {
		for _, t := range srv.GetAllowedTools() {
			if t == nil {
				continue
			}

			if t.Title == title {
				return true
			}
		}
	}

	return false
}

func (s *session) update(update acp.SessionUpdate) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.updates = append(s.updates, update)

	// handle tool call updates
	if update.ToolCall != nil {
		s.toolCallStatusUpdateLocked(&acp.SessionToolCallUpdate{
			Content:       update.ToolCall.Content,
			Kind:          &update.ToolCall.Kind,
			Locations:     update.ToolCall.Locations,
			RawInput:      update.ToolCall.RawInput,
			RawOutput:     update.ToolCall.RawOutput,
			SessionUpdate: update.ToolCall.SessionUpdate,
			Status:        &update.ToolCall.Status,
			Title:         &update.ToolCall.Title,
			ToolCallId:    update.ToolCall.ToolCallId,
		})
	}
	if update.ToolCallUpdate != nil {
		s.toolCallStatusUpdateLocked(update.ToolCallUpdate)
	}
}

// toolCallStatusUpdateLocked updates tool call status. Caller must hold s.mu.
func (s *session) toolCallStatusUpdateLocked(update *acp.SessionToolCallUpdate) {
	call, ok := s.toolCallStatuses[update.ToolCallId]
	if !ok {
		s.toolCallStatuses[update.ToolCallId] = update
		return
	}

	if update.Content != nil {
		call.Content = update.Content
	}

	if update.Kind != nil {
		call.Kind = update.Kind
	}

	if update.Locations != nil {
		call.Locations = update.Locations
	}

	if update.RawInput != nil {
		call.RawInput = update.RawInput
	}

	if update.RawOutput != nil {
		call.RawOutput = update.RawOutput
	}

	if update.Status != nil {
		call.Status = update.Status
	}

	if update.Title != nil {
		call.Title = update.Title
	}
}
