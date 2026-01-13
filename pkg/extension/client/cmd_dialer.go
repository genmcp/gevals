package client

import (
	"context"
	"errors"
	"io"

	"golang.org/x/exp/jsonrpc2"
)

type cmdDialer struct {
	stdin  io.WriteCloser
	stdout io.ReadCloser
}

var _ jsonrpc2.Dialer = &cmdDialer{}

func (d *cmdDialer) Dial(ctx context.Context) (io.ReadWriteCloser, error) {
	return &stdioReadWriteCloser{
		stdin:  d.stdin,
		stdout: d.stdout,
	}, nil
}

type stdioReadWriteCloser struct {
	stdin  io.WriteCloser
	stdout io.ReadCloser
}

var _ io.ReadWriteCloser = &stdioReadWriteCloser{}

func (rwc *stdioReadWriteCloser) Read(data []byte) (int, error) {
	return rwc.stdout.Read(data)
}

func (rwc *stdioReadWriteCloser) Write(data []byte) (int, error) {
	return rwc.stdin.Write(data)
}

func (rwc *stdioReadWriteCloser) Close() error {
	err := rwc.stdin.Close()
	return errors.Join(err, rwc.stdout.Close())
}
