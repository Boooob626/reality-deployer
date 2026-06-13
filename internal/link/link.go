// Package link 生成各组合的客户端分享链接（vless:// / hysteria2://）。
package link

import (
	"encoding/json"
	"fmt"
	"net/url"

	"reality-deployer/internal/combo"
	"reality-deployer/internal/manifest"
)

// Link 是一条导出的客户端链接。
type Link struct {
	Combo string // vless_reality / vless_tls / hysteria2
	Name  string
	URL   string
}

// All 返回 manifest 中所有组合的链接。
func All(m *manifest.Manifest) []Link {
	var out []Link
	for _, c := range m.Combos {
		switch c.Type {
		case combo.TypeVLESSReality:
			if u := VLESSReality(m, c); u != "" {
				out = append(out, Link{c.Type, "VLESS+Vision+REALITY", u})
			}
		case combo.TypeVLESSTLS:
			out = append(out, Link{c.Type, "VLESS+Vision+TLS", VLESSTLS(m, c)})
		case combo.TypeVLESSXHTTP:
			out = append(out, Link{c.Type, "VLESS+XHTTP+TLS", VLESSXHTTP(m, c)})
		case combo.TypeHysteria2:
			out = append(out, Link{c.Type, "Hysteria2", Hysteria2(m, c)})
		}
	}
	return out
}

// VLESSReality 构造 vless:// 链接（REALITY + Vision）。
func VLESSReality(m *manifest.Manifest, c combo.Spec) string {
	rt := m.Reality
	if rt == nil || rt.PublicKey == "" || len(rt.ServerNames) == 0 {
		return ""
	}
	sni := rt.ServerNames[0]
	q := url.Values{}
	q.Set("encryption", "none")
	q.Set("security", "reality")
	q.Set("sni", sni)
	q.Set("fp", "chrome")
	q.Set("pbk", rt.PublicKey)
	q.Set("type", "tcp")
	q.Set("flow", "xtls-rprx-vision")
	if len(rt.ShortIDs) > 1 {
		q.Set("sid", rt.ShortIDs[1])
	}
	return fmt.Sprintf("vless://%s@%s:%d?%s#%s",
		c.UUID, m.Domain, c.Port, q.Encode(), url.PathEscape("reality"))
}

// VLESSTLS 构造 vless:// 链接（TLS + Vision，自有证书）。
func VLESSTLS(m *manifest.Manifest, c combo.Spec) string {
	q := url.Values{}
	q.Set("encryption", "none")
	q.Set("security", "tls")
	q.Set("sni", m.Domain)
	q.Set("fp", "chrome")
	q.Set("type", "tcp")
	q.Set("flow", "xtls-rprx-vision")
	return fmt.Sprintf("vless://%s@%s:%d?%s#%s",
		c.UUID, m.Domain, c.Port, q.Encode(), url.PathEscape("vless-tls"))
}

// VLESSXHTTP 构造 vless:// 链接（XHTTP + TLS）。
func VLESSXHTTP(m *manifest.Manifest, c combo.Spec) string {
	path := c.XHTTPPath
	if path == "" {
		path = "/xhttp"
	}
	mode := c.XHTTPMode
	if mode == "" {
		mode = "auto"
	}
	q := url.Values{}
	q.Set("encryption", "none")
	q.Set("security", "tls")
	q.Set("sni", m.Domain)
	q.Set("fp", "chrome")
	q.Set("type", "xhttp")
	q.Set("host", m.Domain)
	q.Set("path", path)
	q.Set("mode", mode)
	q.Set("extra", xhttpClientExtra())
	return fmt.Sprintf("vless://%s@%s:%d?%s#%s",
		c.UUID, m.Domain, c.Port, q.Encode(), url.PathEscape("xhttp-tls"))
}

func xhttpClientExtra() string {
	extra := map[string]any{
		"xPaddingBytes": "100-1000",
		"xmux": map[string]any{
			"maxConcurrency":   "16-32",
			"maxConnections":   0,
			"cMaxReuseTimes":   0,
			"hMaxRequestTimes": "600-900",
			"hMaxReusableSecs": "1800-3000",
			"hKeepAlivePeriod": 0,
		},
	}
	b, _ := json.Marshal(extra)
	return string(b)
}

// Hysteria2 构造 hysteria2:// 链接（自签证书 → insecure=1）。
func Hysteria2(m *manifest.Manifest, c combo.Spec) string {
	q := url.Values{}
	q.Set("sni", m.Domain)
	q.Set("alpn", "h3")
	q.Set("insecure", "1")
	if c.Hy2Obfs == "salamander" {
		q.Set("obfs", "salamander")
		q.Set("obfs-password", c.Hy2ObfsPwd)
	}
	return fmt.Sprintf("hysteria2://%s@%s:%d?%s#%s",
		url.QueryEscape(c.Hy2Password), m.Domain, c.Port, q.Encode(), url.PathEscape("hysteria2"))
}
