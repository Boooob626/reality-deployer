// Package routing 生成 Xray 路由规则（CN/RU 屏蔽/绕过、私网保护、广告拦截）。
package routing

// Preset 是路由预设类型。
type Preset string

const (
	PresetBlockCN    Preset = "block_cn"     // geosite/geoip cn → block
	PresetBlockCNRU  Preset = "block_cn_ru"  // CN + RU → block（默认）
	PresetBypassCN   Preset = "bypass_cn"    // CN → direct
	PresetBypassCNRU Preset = "bypass_cn_ru" // CN + RU → direct
	PresetNone       Preset = "none"
)

// Rule 对应一条 Xray 路由规则（仅用到的字段子集）。
type Rule map[string]any

// Build 依据预设与广告开关，返回 domainStrategy 与路由规则切片。
// 私网和明文 BT 始终拦截；规则顺序按 Xray "首匹配生效" 语义排好。
func Build(p Preset, adBlock bool) (domainStrategy string, rules []Rule) {
	// 配合 inbound sniffing.routeOnly=true，AsIs 可以做域名/IP 分流且不触发额外 DNS 解析。
	domainStrategy = "AsIs"

	// VPS 不应被当成内网跳板或云元数据探针。
	rules = append(rules, Rule{
		"type":        "field",
		"ip":          []string{"geoip:private"},
		"outboundTag": "block",
	})
	// 降低出口滥用与投诉风险。
	rules = append(rules, Rule{
		"type":        "field",
		"protocol":    []string{"bittorrent"},
		"outboundTag": "block",
	})

	switch p {
	case PresetBlockCN:
		rules = append(rules,
			Rule{"type": "field", "domain": []string{"geosite:cn"}, "outboundTag": "block"},
			Rule{"type": "field", "ip": []string{"geoip:cn"}, "outboundTag": "block"},
		)
	case PresetBlockCNRU:
		rules = append(rules,
			Rule{"type": "field", "domain": []string{"geosite:cn", "domain:ru"}, "outboundTag": "block"},
			Rule{"type": "field", "ip": []string{"geoip:cn", "geoip:ru"}, "outboundTag": "block"},
		)
	case PresetBypassCN:
		rules = append(rules,
			Rule{"type": "field", "domain": []string{"geosite:cn"}, "outboundTag": "direct"},
			Rule{"type": "field", "ip": []string{"geoip:cn"}, "outboundTag": "direct"},
		)
	case PresetBypassCNRU:
		rules = append(rules,
			Rule{"type": "field", "domain": []string{"geosite:cn", "domain:ru"}, "outboundTag": "direct"},
			Rule{"type": "field", "ip": []string{"geoip:cn", "geoip:ru"}, "outboundTag": "direct"},
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
