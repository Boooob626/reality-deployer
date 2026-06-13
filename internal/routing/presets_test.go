package routing

import "testing"

func TestBuildBlockCNEmitsBlockRules(t *testing.T) {
	_, rules := Build(PresetBlockCN, false)
	var hasPrivateDirect, hasCNBlock bool
	for _, r := range rules {
		tags, _ := r["outboundTag"].(string)
		if tags == "direct" {
			if ips, ok := r["ip"].([]string); ok {
				for _, ip := range ips {
					if ip == "geoip:private" {
						hasPrivateDirect = true
					}
				}
			}
		}
		if tags == "block" {
			if doms, ok := r["domain"].([]string); ok {
				for _, d := range doms {
					if d == "geosite:cn" {
						hasCNBlock = true
					}
				}
			}
		}
	}
	if !hasPrivateDirect {
		t.Error("缺少 geoip:private → direct")
	}
	if !hasCNBlock {
		t.Error("缺少 geosite:cn → block")
	}
}

func TestBuildBypassCNDirectsCN(t *testing.T) {
	_, rules := Build(PresetBypassCN, false)
	for _, r := range rules {
		if doms, ok := r["domain"].([]string); ok {
			for _, d := range doms {
				if d == "geosite:cn" {
					if r["outboundTag"] != "direct" {
						t.Errorf("bypass_cn: geosite:cn 应为 direct, got %v", r["outboundTag"])
					}
					return
				}
			}
		}
	}
}

func TestBuildAdBlock(t *testing.T) {
	_, rules := Build(PresetNone, true)
	var hasAdsBlock bool
	for _, r := range rules {
		if r["outboundTag"] == "block" {
			if doms, ok := r["domain"].([]string); ok {
				for _, d := range doms {
					if d == "geosite:category-ads-all" {
						hasAdsBlock = true
					}
				}
			}
		}
	}
	if !hasAdsBlock {
		t.Error("adBlock 应产生 category-ads-all → block")
	}
}
