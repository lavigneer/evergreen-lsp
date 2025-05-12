package lsp

import (
	"context"
	"log"
	"os"

	"github.com/sourcegraph/jsonrpc2"
)

type LSP struct {
	handler jsonrpc2.Handler
	logger  *log.Logger
}

func New(handler jsonrpc2.Handler, logger *log.Logger) *LSP {
	return &LSP{handler, logger}
}

func (l LSP) Start(ctx context.Context) <-chan struct{} {
	return jsonrpc2.NewConn(
		ctx,
		jsonrpc2.NewBufferedStream(stdrwc{}, jsonrpc2.VSCodeObjectCodec{}),
		l.handler,
		jsonrpc2.LogMessages(l.logger),
	).DisconnectNotify()
}

type stdrwc struct{}

func (stdrwc) Read(p []byte) (int, error) {
	return os.Stdin.Read(p)
}

func (stdrwc) Write(p []byte) (int, error) {
	return os.Stdout.Write(p)
}

func (stdrwc) Close() error {
	if err := os.Stdin.Close(); err != nil {
		return err
	}

	return os.Stdout.Close()
}
