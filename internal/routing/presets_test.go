package routing

import "testing"

func TestBuildBlockCNEmitsBlockRules(t *testing.T) {
	strategy, rules := Build(PresetBlockCN, false)
	if strategy != "AsIs" {
		t.Fatalf("domainStrategy=%s, want AsIs", strategy)
	}
	var hasPrivateBlock, hasBTBlock, hasCNBlock bool
	for _, r := range rules {
		tags, _ := r["outboundTag"].(string)
		if tags == "block" {
			if ips, ok := r["ip"].([]string); ok {
				for _, ip := range ips {
					if ip == "geoip:private" {
						hasPrivateBlock = true
					}
				}
			}
			if doms, ok := r["domain"].([]string); ok {
				for _, d := range doms {
					if d == "geosite:cn" {
						hasCNBlock = true
					}
				}
			}
			if protos, ok := r["protocol"].([]string); ok {
				for _, p := range protos {
					if p == "bittorrent" {
						hasBTBlock = true
					}
				}
			}
		}
	}
	if !hasPrivateBlock {
		t.Error("缺少 geoip:private → block")
	}
	if !hasBTBlock {
		t.Error("缺少 bittorrent → block")
	}
	if !hasCNBlock {
		t.Error("缺少 geosite:cn → block")
	}
}

func TestBuildBlockCNRUBlocksRU(t *testing.T) {
	_, rules := Build(PresetBlockCNRU, false)
	var hasRUDomain, hasRUIP bool
	for _, r := range rules {
		if r["outboundTag"] != "block" {
			continue
		}
		if doms, ok := r["domain"].([]string); ok {
			for _, d := range doms {
				if d == "domain:ru" {
					hasRUDomain = true
				}
			}
		}
		if ips, ok := r["ip"].([]string); ok {
			for _, ip := range ips {
				if ip == "geoip:ru" {
					hasRUIP = true
				}
			}
		}
	}
	if !hasRUDomain || !hasRUIP {
		t.Fatalf("block_cn_ru 缺少 RU domain/ip 规则: domain=%v ip=%v", hasRUDomain, hasRUIP)
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
