// Package combo 描述协议组合并生成对应的 Xray inbound。
// 每个组合实现为一个生成函数；config 渲染器收集这些 inbound 拼成最终 config.json。
package combo

import (
	"fmt"

	"reality-deployer/internal/paths"
	"reality-deployer/internal/reality"
)

// 组合类型常量。
const (
	TypeVLESSReality = "vless_reality" // VLESS + Vision + REALITY/TCP（主力）
	TypeVLESSTLS     = "vless_tls"     // VLESS + Vision + TLS（自有 ACME 证书）
	TypeVLESSXHTTP   = "vless_xhttp"   // VLESS + XHTTP + TLS（XMUX-capable）
	TypeHysteria2    = "hysteria2"
)

// Spec 描述一个已决策的组合；同时是 manifest 的持久化形态。
type Spec struct {
	Type string `json:"type"`
	UUID string `json:"uuid"`
	Port int    `json:"port"`
	Flow string `json:"flow,omitempty"` // xtls-rprx-vision

	// XHTTP 专用
	XHTTPPath string `json:"xhttp_path,omitempty"`
	XHTTPMode string `json:"xhttp_mode,omitempty"` // auto / packet-up / stream-up / stream-one

	// Hysteria2 专用
	Hy2Password string `json:"hy2_password,omitempty"`
	Hy2Obfs     string `json:"hy2_obfs,omitempty"` // "salamander" 或空
	Hy2ObfsPwd  string `json:"hy2_obfs_pwd,omitempty"`
	Hy2UpMBps   int    `json:"hy2_up_mbps,omitempty"`
	Hy2DownMBps int    `json:"hy2_down_mbps,omitempty"`
	Hy2PortHop  string `json:"hy2_port_hop,omitempty"` // ufw 放行范围 "20000:50000"
}

// Inbound 是一个 Xray inbound 对象（渲染到 JSON）。
type Inbound = map[string]any

func sniffing() map[string]any {
	return map[string]any{
		"enabled":      true,
		"destOverride": []string{"http", "tls", "quic"},
		"routeOnly":    true,
	}
}

// VLESSReality 生成 VLESS + Vision + REALITY/TCP 入站。
// REALITY 经握手借用目标站点证书，Xray 无需持有证书文件。
func VLESSReality(s Spec, rt *reality.Target) (Inbound, error) {
	if rt == nil {
		return nil, fmt.Errorf("REALITY 组合缺少 target 配置")
	}
	return map[string]any{
		"tag":      "in-vless-reality",
		"listen":   "0.0.0.0",
		"port":     s.Port,
		"protocol": "vless",
		"settings": map[string]any{
			"clients": []map[string]any{
				{"id": s.UUID, "flow": "xtls-rprx-vision"},
			},
			"decryption": "none",
		},
		"streamSettings": map[string]any{
			"network":  "tcp",
			"security": "reality",
			"realitySettings": map[string]any{
				"show":        false,
				"target":      rt.Target,
				"xver":        0,
				"serverNames": rt.ServerNames,
				"privateKey":  rt.PrivateKey,
				"shortIds":    rt.ShortIDs,
			},
		},
		"sniffing": sniffing(),
	}, nil
}

// VLESSTLS 生成 VLESS + Vision + TLS 入站，复用 Angie ACME 签发的域名证书。
func VLESSTLS(s Spec, domain string) Inbound {
	return map[string]any{
		"tag":      "in-vless-tls",
		"listen":   "0.0.0.0",
		"port":     s.Port,
		"protocol": "vless",
		"settings": map[string]any{
			"clients": []map[string]any{
				{"id": s.UUID, "flow": "xtls-rprx-vision"},
			},
			"decryption": "none",
		},
		"streamSettings": map[string]any{
			"network":  "tcp",
			"security": "tls",
			"tlsSettings": map[string]any{
				"serverName": domain,
				"minVersion": "1.2",
				"certificates": []map[string]any{
					{
						"certificateFile": paths.DomainCert(domain),
						"keyFile":         paths.DomainKey(domain),
					},
				},
			},
		},
		"sniffing": sniffing(),
	}
}

// VLESSXHTTP 生成 VLESS + XHTTP + TLS 入站。
func VLESSXHTTP(s Spec, domain string) Inbound {
	path := s.XHTTPPath
	if path == "" {
		path = "/xhttp"
	}
	mode := s.XHTTPMode
	if mode == "" {
		mode = "auto"
	}
	return map[string]any{
		"tag":      "in-vless-xhttp",
		"listen":   "0.0.0.0",
		"port":     s.Port,
		"protocol": "vless",
		"settings": map[string]any{
			"clients": []map[string]any{
				{"id": s.UUID},
			},
			"decryption": "none",
		},
		"streamSettings": map[string]any{
			"network":  "xhttp",
			"security": "tls",
			"tlsSettings": map[string]any{
				"serverName": domain,
				"minVersion": "1.2",
				"alpn":       []string{"h2", "http/1.1"},
				"certificates": []map[string]any{
					{
						"certificateFile": paths.DomainCert(domain),
						"keyFile":         paths.DomainKey(domain),
					},
				},
			},
			"xhttpSettings": map[string]any{
				"host": domain,
				"path": path,
				"mode": mode,
				"extra": map[string]any{
					"xPaddingBytes":        "100-1000",
					"scMaxBufferedPosts":   30,
					"scStreamUpServerSecs": "20-80",
				},
			},
		},
		"sniffing": sniffing(),
	}
}

// Hysteria2 生成 Hysteria2 入站（QUIC/UDP，自签证书）。
// 结构依据 XTLS/Xray-core PR #5679 的权威示例：
//   - protocol = "hysteria"（version:2 写在 settings 与 hysteriaSettings）
//   - network  = "hysteria"
//   - salamander obfs 在 streamSettings.finalmask.udp
func Hysteria2(s Spec) Inbound {
	stream := map[string]any{
		"network":          "hysteria",
		"hysteriaSettings": map[string]any{"version": 2},
		"security":         "tls",
		"tlsSettings": map[string]any{
			"alpn": []string{"h3"},
			"certificates": []map[string]any{
				{
					"certificateFile": paths.Hy2Cert,
					"keyFile":         paths.Hy2Key,
				},
			},
		},
	}
	if s.Hy2Obfs == "salamander" && s.Hy2ObfsPwd != "" {
		stream["finalmask"] = map[string]any{
			"udp": []map[string]any{
				{
					"type":     "salamander",
					"settings": map[string]any{"password": s.Hy2ObfsPwd},
				},
			},
		}
	}
	return map[string]any{
		"tag":      "in-hysteria2",
		"listen":   "0.0.0.0",
		"port":     s.Port,
		"protocol": "hysteria",
		"settings": map[string]any{
			"version": 2,
			"clients": []map[string]any{
				{"auth": s.Hy2Password},
			},
		},
		"streamSettings": stream,
		"sniffing":       sniffing(),
	}
}

// InboundFor 根据 spec 类型分发到对应生成器。
func InboundFor(s Spec, rt *reality.Target, domain string) (Inbound, error) {
	switch s.Type {
	case TypeVLESSReality:
		return VLESSReality(s, rt)
	case TypeVLESSTLS:
		return VLESSTLS(s, domain), nil
	case TypeVLESSXHTTP:
		return VLESSXHTTP(s, domain), nil
	case TypeHysteria2:
		return Hysteria2(s), nil
	default:
		return nil, fmt.Errorf("未知组合类型: %q", s.Type)
	}
}
