// Package routing 生成 Xray 路由规则（CN 屏蔽/绕过、私网直连、广告拦截）。
package routing

// Preset 是路由预设类型。
type Preset string

const (
	PresetBlockCN  Preset = "block_cn"  // geosite/geoip cn → block（默认）
	PresetBypassCN Preset = "bypass_cn" // cn → direct
	PresetNone     Preset = "none"
)

// Rule 对应一条 Xray 路由规则（仅用到的字段子集）。
type Rule map[string]any

// Build 依据预设与广告开关，返回 domainStrategy 与路由规则切片。
// 私网始终直连；规则顺序按 Xray "首匹配生效" 语义排好。
func Build(p Preset, adBlock bool) (domainStrategy string, rules []Rule) {
	domainStrategy = "IPIfNonMatch"

	// 私网/LAN 始终直连
	rules = append(rules, Rule{
		"type":        "field",
		"ip":          []string{"geoip:private"},
		"outboundTag": "direct",
	})

	switch p {
	case PresetBlockCN:
		rules = append(rules,
			Rule{"type": "field", "domain": []string{"geosite:cn"}, "outboundTag": "block"},
			Rule{"type": "field", "ip": []string{"geoip:cn"}, "outboundTag": "block"},
		)
	case PresetBypassCN:
		rules = append(rules,
			Rule{"type": "field", "domain": []string{"geosite:cn"}, "outboundTag": "direct"},
			Rule{"type": "field", "ip": []string{"geoip:cn"}, "outboundTag": "direct"},
		)
	case PresetNone:
		// 不附加区域规则
	}

	if adBlock {
		rules = append(rules, Rule{
			"type":        "field",
			"domain":      []string{"geosite:category-ads-all"},
			"outboundTag": "block",
		})
	}

	return domainStrategy, rules
}
