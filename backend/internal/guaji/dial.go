package guaji

import (
	"context"
	"fmt"
	"net"
	"sort"
	"strings"
	"time"
)

const (
	perIPDialTimeout = 2 * time.Second
	maxDialIPs       = 6
)

// dialContextPreferHealthy 解析全部 A/AAAA，按短超时逐个拨号。
// CDN 常返回部分死 IP（本环境曾见 121.127.246.189 不通、.176/.171 可用），
// 默认 net.Dialer 只试第一个 IP 会长时间卡死 periods/下单。
func dialContextPreferHealthy(ctx context.Context, network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	if host == "" {
		return nil, fmt.Errorf("empty dial host")
	}
	// 已是字面量 IP：直接拨。
	if ip := net.ParseIP(host); ip != nil {
		d := &net.Dialer{Timeout: perIPDialTimeout, KeepAlive: 30 * time.Second}
		return d.DialContext(ctx, network, addr)
	}

	lookupCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	ips, err := net.DefaultResolver.LookupIPAddr(lookupCtx, host)
	cancel()
	if err != nil {
		return nil, err
	}
	if len(ips) == 0 {
		return nil, fmt.Errorf("no IPs for %s", host)
	}

	// 稳定顺序：先 IPv4，再按字符串，避免每次随机撞死 IP。
	sort.SliceStable(ips, func(i, j int) bool {
		a, b := ips[i].IP, ips[j].IP
		a4, b4 := a.To4() != nil, b.To4() != nil
		if a4 != b4 {
			return a4
		}
		return a.String() < b.String()
	})
	if len(ips) > maxDialIPs {
		ips = ips[:maxDialIPs]
	}

	var errs []string
	for _, ipa := range ips {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		ip := ipa.IP
		if ip == nil {
			continue
		}
		target := net.JoinHostPort(ip.String(), port)
		dctx, dcancel := context.WithTimeout(ctx, perIPDialTimeout)
		d := &net.Dialer{Timeout: perIPDialTimeout, KeepAlive: 30 * time.Second}
		conn, derr := d.DialContext(dctx, network, target)
		dcancel()
		if derr == nil {
			return conn, nil
		}
		errs = append(errs, fmt.Sprintf("%s: %v", target, derr))
	}
	if len(errs) == 0 {
		return nil, fmt.Errorf("dial %s: no usable address", addr)
	}
	return nil, fmt.Errorf("dial %s: all IPs failed (%s)", addr, strings.Join(errs, "; "))
}
