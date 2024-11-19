// Copyright (c) 2024 RoseLoverX

package transport

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/pkg/errors"
)

type tcpConn struct {
	reader  *Reader
	conn    *net.TCPConn
	timeout time.Duration
}

type TCPConnConfig struct {
	Ctx     context.Context
	Host    string
	IpV6    bool
	Timeout time.Duration
	Socks   *url.URL
}

func formatIPv6WithPort(ipv6WithPort string) (string, error) {
	// Find the last colon to split the port
	lastColon := strings.LastIndex(ipv6WithPort, ":")
	if lastColon == -1 {
		return "", fmt.Errorf("invalid IPv6 address with port")
	}

	// Extract the address and port
	address := ipv6WithPort[:lastColon]
	port := ipv6WithPort[lastColon+1:]

	// Validate that the port is numeric
	if len(port) == 0 || strings.ContainsAny(port, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ") {
		return "", fmt.Errorf("invalid port number")
	}

	// Return the formatted address
	return fmt.Sprintf("[%s]:%s", address, port), nil
}

func NewTCP(cfg TCPConnConfig) (Conn, error) {
	if cfg.Socks != nil && cfg.Socks.Host != "" {
		return newSocksTCP(cfg)
	}
	tcpPrefix := "tcp"
	if cfg.IpV6 {
		fmt.Println("ipv6")
		tcpPrefix = "tcp6"

		cfg.Host, _ = formatIPv6WithPort(cfg.Host)

	}

	// cfg.Host = "[2001:4860:4860::8888]:443"

	// its in 2001:0b28:f23f:f005:0000:0000:0000:000a:443, format
	// change it to [2001:0b28:f23f:f005::a]:443

	tcpAddr, err := net.ResolveTCPAddr(tcpPrefix, cfg.Host)
	if err != nil {
		return nil, errors.Wrap(err, "resolving tcp")
	}
	conn, err := net.DialTCP(tcpPrefix, nil, tcpAddr)
	if err != nil {
		return nil, errors.Wrap(err, "dialing tcp")
	}

	return &tcpConn{
		reader:  NewReader(cfg.Ctx, conn),
		conn:    conn,
		timeout: cfg.Timeout,
	}, nil
}

func newSocksTCP(cfg TCPConnConfig) (Conn, error) {
	conn, err := dialProxy(cfg.Socks, cfg.Host)
	if err != nil {
		return nil, err
	}
	return &tcpConn{
		reader:  NewReader(cfg.Ctx, conn),
		conn:    conn.(*net.TCPConn),
		timeout: cfg.Timeout,
	}, nil
}

func (t *tcpConn) Close() error {
	return t.conn.Close()
}

func (t *tcpConn) Write(b []byte) (int, error) {
	return t.conn.Write(b)
}

func (t *tcpConn) Read(b []byte) (int, error) {
	if t.timeout > 0 {
		err := t.conn.SetReadDeadline(time.Now().Add(t.timeout))
		if err != nil {
			return 0, errors.Wrap(err, "setting read deadline")
		}
	}

	n, err := t.reader.Read(b)
	if err != nil {
		if e, ok := err.(*net.OpError); ok || err == io.ErrClosedPipe {
			if e.Err.Error() == "i/o timeout" || err == io.ErrClosedPipe {
				return 0, errors.Wrap(err, "required to reconnect!")
			}
		}
		switch err {
		case io.EOF, context.Canceled:
			return 0, err
		default:
			return 0, errors.Wrap(err, "unexpected error")
		}
	}
	return n, nil
}
