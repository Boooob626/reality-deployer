package wizard

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net"
	"regexp"
	"time"
)

var emailRe = regexp.MustCompile(`^[A-Za-z0-9._%+\-]+@[A-Za-z0-9](?:[A-Za-z0-9-]{0,61}[A-Za-z0-9])?(?:\.[A-Za-z0-9](?:[A-Za-z0-9-]{0,61}[A-Za-z0-9])?)+$`)
var domainRe = regexp.MustCompile(`(?i)^([a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$`)

// validEmail 校验邮箱格式。
func validEmail(s string) bool {
	return emailRe.MatchString(s)
}

// validDomain 校验会被写入 Angie/Xray 配置的域名，避免无效域名或配置注入。
func validDomain(s string) bool {
	return len(s) <= 253 && domainRe.MatchString(s)
}

// detectPublicIP 枚举网卡，返回首个非回环、非私网的 IPv4（VPS 常见情形）。
// 找不到公网时回退到首个非回环 IPv4；再找不到返回空串（向导让用户手填）。
func detectPublicIP() string {
	ifs, err := net.Interfaces()
	if err != nil {
		return ""
	}
	var fallback string
	for _, ifc := range ifs {
		if ifc.Flags&net.FlagUp == 0 || ifc.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := ifc.Addrs()
		if err != nil {
			continue
		}
		for _, a := range addrs {
			ipNet, ok := a.(*net.IPNet)
			if !ok {
				continue
			}
			ip := ipNet.IP
			if ip.To4() == nil || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
				continue
			}
			str := ip.String()
			if fallback == "" {
				fallback = str
			}
			if !ip.IsPrivate() {
				return str
			}
		}
	}
	return fallback
}

// resolveDomain 解析域名的 IPv4 列表。
func resolveDomain(domain string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	ips, err := net.DefaultResolver.LookupIP(ctx, "ip4", domain)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(ips))
	for _, ip := range ips {
		out = append(out, ip.String())
	}
	return out, nil
}

// portFreeUDP 做尽力而为的 UDP 端口占用检测。
func portFreeUDP(port int) bool {
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

// genUUID 生成 RFC 4122 v4 UUID。
func genUUID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// genPassword 生成随机密码（hex）。
func genPassword(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return fmt.Sprintf("%x", b)
}

func genPathToken(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}
