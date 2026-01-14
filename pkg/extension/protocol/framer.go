package protocol

import (
	"bufio"
	"context"
	"io"
	"sync"

	"golang.org/x/exp/jsonrpc2"
)

func NewlineFramer() jsonrpc2.Framer {
	return &newlineFramer{}
}

type newlineFramer struct{}

func (f *newlineFramer) Reader(r io.Reader) jsonrpc2.Reader {
	return &newlineReader{scanner: bufio.NewScanner(r)}
}

func (f *newlineFramer) Writer(w io.Writer) jsonrpc2.Writer {
	return &newlineWriter{w: w}
}

type scanResult struct {
	data []byte
	err  error
}

// newlineReader reads newline-delimited JSON-RPC messages.
// It uses a persistent reader goroutine to avoid goroutine leaks when
// context is canceled during a blocking read.
type newlineReader struct {
	scanner  *bufio.Scanner
	resultCh chan scanResult
	once     sync.Once
}

// startReader starts the persistent reader goroutine that continuously
// reads from the scanner. This goroutine only exits when the scanner
// encounters an error or EOF, ensuring no goroutine leaks on context
// cancellation.
func (r *newlineReader) startReader() {
	r.once.Do(func() {
		r.resultCh = make(chan scanResult)
		go func() {
			defer close(r.resultCh)
			for r.scanner.Scan() {
				// Make a copy of the data since scanner reuses the buffer
				data := make([]byte, len(r.scanner.Bytes()))
				copy(data, r.scanner.Bytes())
				r.resultCh <- scanResult{data: data}
			}
			// Send the final error (nil for clean EOF)
			r.resultCh <- scanResult{err: r.scanner.Err()}
		}()
	})
}

func (r *newlineReader) Read(ctx context.Context) (jsonrpc2.Message, int64, error) {
	r.startReader()

	select {
	case <-ctx.Done():
		return nil, 0, ctx.Err()
	case result, ok := <-r.resultCh:
		if !ok {
			// Channel closed, reader goroutine has exited
			return nil, 0, io.EOF
		}
		if result.err != nil {
			return nil, 0, result.err
		}
		if result.data == nil {
			return nil, 0, io.EOF
		}

		msg, err := jsonrpc2.DecodeMessage(result.data)
		if err != nil {
			return nil, 0, err
		}
		return msg, int64(len(result.data)), nil
	}
}

type newlineWriter struct {
	mu sync.Mutex
	w  io.Writer
}

func (w *newlineWriter) Write(ctx context.Context, msg jsonrpc2.Message) (int64, error) {
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	default:
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	data, err := jsonrpc2.EncodeMessage(msg)
	if err != nil {
		return 0, err
	}
	data = append(data, '\n')
	n, err := w.w.Write(data)
	return int64(n), err
}
