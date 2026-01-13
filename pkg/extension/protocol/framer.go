package protocol

import (
	"bufio"
	"context"
	"io"

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

type newlineReader struct {
	scanner *bufio.Scanner
}

func (r *newlineReader) Read(ctx context.Context) (jsonrpc2.Message, int64, error) {
	select {
	case <-ctx.Done():
		return nil, 0, ctx.Err()
	default:
	}

	if !r.scanner.Scan() {
		if err := r.scanner.Err(); err != nil {
			return nil, 0, err
		}
		return nil, 0, io.EOF
	}

	data := r.scanner.Bytes()
	msg, err := jsonrpc2.DecodeMessage(data)
	if err != nil {
		return nil, 0, err
	}

	return msg, int64(len(data)), nil
}

type newlineWriter struct {
	w io.Writer
}

func (w *newlineWriter) Write(ctx context.Context, msg jsonrpc2.Message) (int64, error) {
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	default:
	}

	data, err := jsonrpc2.EncodeMessage(msg)
	if err != nil {
		return 0, err
	}
	data = append(data, '\n')
	n, err := w.w.Write(data)
	return int64(n), err
}
