// Package config 把 manifest 渲染成各组件的配置文件与 apply.env / ufw 脚本。
package config

import (
	"encoding/json"
	"fmt"

	"reality-deployer/internal/combo"
	"reality-deployer/internal/manifest"
	"reality-deployer/internal/routing"
)

// Xray 返回完整的 Xray config.json 字节。
func Xray(m *manifest.Manifest) ([]byte, error) {
	inbounds := make([]combo.Inbound, 0, len(m.Combos))
	for _, c := range m.Combos {
		in, err := combo.InboundFor(c, m.Reality, m.Domain)
		if err != nil {
			return nil, fmt.Errorf("组合 %s: %w", c.Type, err)
		}
		inbounds = append(inbounds, in)
	}

	domainStrategy, rules := routing.Build(routing.Preset(m.Routing.Preset), m.Routing.AdBlock)

	cfg := map[string]any{
		"log": map[string]any{
			"loglevel": "warning",
		},
		"inbounds": inbounds,
		"policy": map[string]any{
			"levels": map[string]any{
				"0": map[string]any{
					"handshake":         4,
					"connIdle":          300,
					"uplinkOnly":        0,
					"downlinkOnly":      0,
					"statsUserUplink":   false,
					"statsUserDownlink": false,
					"statsUserOnline":   false,
				},
			},
			"system": map[string]any{
				"statsInboundUplink":    false,
				"statsInboundDownlink":  false,
				"statsOutboundUplink":   false,
				"statsOutboundDownlink": false,
			},
		},
		"outbounds": []map[string]any{
			{
				"tag":      "direct",
				"protocol": "freedom",
				"settings": map[string]any{"domainStrategy": "AsIs"},
			},
			{
				"tag":      "block",
				"protocol": "blackhole",
				"settings": map[string]any{"response": map[string]any{"type": "http"}},
			},
		},
		"routing": map[string]any{
			"domainStrategy": domainStrategy,
			"rules":          rules,
		},
	}
	return json.MarshalIndent(cfg, "", "  ")
}
