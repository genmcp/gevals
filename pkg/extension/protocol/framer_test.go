package protocol

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/jsonrpc2"
)

func TestNewlineReader_Read(t *testing.T) {
	tt := map[string]struct {
		input       string
		expectedID  jsonrpc2.ID
		expectedLen int64
		expectErr   error
	}{
		"valid request message": {
			input:       `{"jsonrpc":"2.0","id":1,"method":"test"}` + "\n",
			expectedID:  jsonrpc2.Int64ID(1),
			expectedLen: 40,
			expectErr:   nil,
		},
		"valid notification message": {
			input:       `{"jsonrpc":"2.0","method":"notify"}` + "\n",
			expectedID:  jsonrpc2.ID{},
			expectedLen: 35,
			expectErr:   nil,
		},
		"empty input returns EOF": {
			input:       "",
			expectedID:  jsonrpc2.ID{},
			expectedLen: 0,
			expectErr:   io.EOF,
		},
		"invalid JSON returns error": {
			input:       `{invalid json}` + "\n",
			expectedID:  jsonrpc2.ID{},
			expectedLen: 0,
			expectErr:   errors.New("json error"),
		},
	}

	for tn, tc := range tt {
		t.Run(tn, func(t *testing.T) {
			framer := NewlineFramer()
			reader := framer.Reader(bytes.NewReader([]byte(tc.input)))

			msg, n, err := reader.Read(context.Background())
			if tc.expectErr != nil {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.expectedLen, n)
			assert.NotNil(t, msg)
		})
	}
}

func TestNewlineReader_Read_ContextCancellation(t *testing.T) {
	framer := NewlineFramer()
	reader := framer.Reader(bytes.NewReader([]byte(`{"jsonrpc":"2.0","id":1,"method":"test"}` + "\n")))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, _, err := reader.Read(ctx)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestNewlineReader_Read_MultipleMessages(t *testing.T) {
	input := `{"jsonrpc":"2.0","id":1,"method":"first"}` + "\n" +
		`{"jsonrpc":"2.0","id":2,"method":"second"}` + "\n"

	framer := NewlineFramer()
	reader := framer.Reader(bytes.NewReader([]byte(input)))

	// Read first message
	msg1, _, err := reader.Read(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, msg1)

	// Read second message
	msg2, _, err := reader.Read(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, msg2)

	// Third read should return EOF
	_, _, err = reader.Read(context.Background())
	assert.ErrorIs(t, err, io.EOF)
}

func TestNewlineWriter_Write(t *testing.T) {
	tt := map[string]struct {
		msg       jsonrpc2.Message
		expectErr bool
	}{
		"write request message": {
			msg:       mustNewCall(t, 1, "test", nil),
			expectErr: false,
		},
		"write notification message": {
			msg:       mustNewNotification(t, "notify", nil),
			expectErr: false,
		},
	}

	for tn, tc := range tt {
		t.Run(tn, func(t *testing.T) {
			var buf bytes.Buffer
			framer := NewlineFramer()
			writer := framer.Writer(&buf)

			n, err := writer.Write(context.Background(), tc.msg)
			if tc.expectErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Greater(t, n, int64(0))
			assert.True(t, bytes.HasSuffix(buf.Bytes(), []byte("\n")), "output should end with newline")
		})
	}
}

func TestNewlineWriter_Write_ContextCancellation(t *testing.T) {
	var buf bytes.Buffer
	framer := NewlineFramer()
	writer := framer.Writer(&buf)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	msg := mustNewCall(t, 1, "test", nil)
	_, err := writer.Write(ctx, msg)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestRoundTrip(t *testing.T) {
	tt := map[string]struct {
		method string
		id     int64
	}{
		"simple request": {
			method: "test",
			id:     1,
		},
		"request with special characters in method": {
			method: "namespace/method",
			id:     42,
		},
	}

	for tn, tc := range tt {
		t.Run(tn, func(t *testing.T) {
			var buf bytes.Buffer
			framer := NewlineFramer()

			// Write a message
			writer := framer.Writer(&buf)
			originalMsg := mustNewCall(t, tc.id, tc.method, nil)
			_, err := writer.Write(context.Background(), originalMsg)
			require.NoError(t, err)

			// Read it back
			reader := framer.Reader(&buf)
			readMsg, _, err := reader.Read(context.Background())
			require.NoError(t, err)

			// Verify the message matches
			req, ok := readMsg.(*jsonrpc2.Request)
			require.True(t, ok, "expected request message")
			assert.Equal(t, tc.method, req.Method)
			assert.Equal(t, jsonrpc2.Int64ID(tc.id), req.ID)
		})
	}
}

// Helper functions

func mustNewCall(t *testing.T, id int64, method string, params any) *jsonrpc2.Request {
	t.Helper()
	req, err := jsonrpc2.NewCall(jsonrpc2.Int64ID(id), method, params)
	require.NoError(t, err)
	return req
}

func mustNewNotification(t *testing.T, method string, params any) *jsonrpc2.Request {
	t.Helper()
	notif, err := jsonrpc2.NewNotification(method, params)
	require.NoError(t, err)
	return notif
}
