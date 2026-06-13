package manifest

import (
	"path/filepath"
	"testing"

	"reality-deployer/internal/combo"
)

func TestBuildFirewall(t *testing.T) {
	combos := []combo.Spec{
		{Type: combo.TypeVLESSReality, Port: 443},
		{Type: combo.TypeVLESSXHTTP, Port: 443},
		{Type: combo.TypeHysteria2, Port: 36712, Hy2PortHop: "20000:50000"},
	}
	fw := BuildFirewall(combos, 2222)
	wantPorts := map[int]bool{2222: false, 80: false, 443: false, 36712: false}
	var sawHop bool
	var count443 int
	for _, r := range fw.Rules {
		if r.Port != 0 {
			wantPorts[r.Port] = true
		}
		if r.Port == 443 && r.Proto == "tcp" {
			count443++
		}
		if r.Range == "20000:50000" {
			sawHop = true
		}
	}
	for p, saw := range wantPorts {
		if !saw {
			t.Errorf("缺少端口 %d 的放行规则", p)
		}
	}
	if !sawHop {
		t.Error("缺少 hysteria2 端口跳跃区间规则")
	}
	if count443 != 1 {
		t.Fatalf("443/tcp 应只放行一次，got %d", count443)
	}
}

func TestNewDefaults(t *testing.T) {
	m := New()
	if m.Routing.Preset != "block_cn_ru" {
		t.Fatalf("默认路由应屏蔽 CN/RU，got %q", m.Routing.Preset)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "manifest.json")
	m1 := New()
	m1.Domain = "example.com"
	m1.Email = "a@example.com"
	m1.Combos = []combo.Spec{{Type: combo.TypeVLESSReality, UUID: "u-1", Port: 443}}
	if err := m1.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if !Exists(path) {
		t.Fatal("Exists 应为 true")
	}
	m2, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if m2.Domain != "example.com" || m2.Schema != SchemaVersion {
		t.Errorf("round-trip 基本字段不符: %+v", m2)
	}
	if len(m2.Combos) != 1 || m2.Combos[0].UUID != "u-1" {
		t.Errorf("round-trip combos 不符: %+v", m2.Combos)
	}
}
