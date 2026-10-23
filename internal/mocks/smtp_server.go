package mocks

import (
	"context"
	"io"
	"net"
	"net/textproto"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// NewSMTPServer is an unauthenticated local SMTP sink for tests only.
func NewSMTPServer(t *testing.T) (string, string, <-chan []byte) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	host, port, err := net.SplitHostPort(listener.Addr().String())
	require.NoError(t, err)
	messages := make(chan []byte, 10)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			connection, err := listener.Accept()
			if err != nil {
				return
			}
			serveSMTP(ctx, connection, messages)
		}
	}()
	t.Cleanup(func() { cancel(); listener.Close(); <-done })
	return host, port, messages
}

func serveSMTP(ctx context.Context, connection net.Conn, messages chan<- []byte) {
	defer connection.Close()
	stop := context.AfterFunc(ctx, func() { connection.Close() })
	defer stop()
	protocol := textproto.NewConn(connection)
	if err := protocol.PrintfLine("220 localhost test SMTP"); err != nil {
		return
	}
	for {
		line, err := protocol.ReadLine()
		if err != nil {
			return
		}
		command := strings.ToUpper(strings.SplitN(line, " ", 2)[0])
		switch command {
		case "EHLO", "HELO", "MAIL", "RCPT", "RSET":
			err = protocol.PrintfLine("250 OK")
		case "DATA":
			if err = protocol.PrintfLine("354 Send message"); err != nil {
				return
			}
			body, readErr := io.ReadAll(protocol.DotReader())
			if readErr != nil {
				return
			}
			select {
			case messages <- body:
			case <-ctx.Done():
				return
			}
			err = protocol.PrintfLine("250 Accepted")
		case "QUIT":
			_ = protocol.PrintfLine("221 Bye")
			return
		default:
			err = protocol.PrintfLine("502 Unsupported")
		}
		if err != nil {
			return
		}
	}
}
