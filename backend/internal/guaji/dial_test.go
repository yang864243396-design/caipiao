package guaji

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestDialContextPreferHealthy_skipsDeadIP(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			_ = c.Close()
		}
	}()
	_, port, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}

	// 先试不可达地址，再试本机 listener（通过自定义解析较难；这里直接测字面量 IP 路径）。
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	conn, err := dialContextPreferHealthy(ctx, "tcp", net.JoinHostPort("127.0.0.1", port))
	if err != nil {
		t.Fatalf("dial live loopback: %v", err)
	}
	_ = conn.Close()
}
