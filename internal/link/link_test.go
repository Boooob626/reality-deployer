package link

import (
	"strings"
	"testing"

	"reality-deployer/internal/combo"
	"reality-deployer/internal/manifest"
	"reality-deployer/internal/reality"
)

func TestAllLinks(t *testing.T) {
	ban, err := reality.LoadBanlist()
	if err != nil {
		t.Fatal(err)
	}
	rt, err := reality.Resolve(reality.SourceOwnDomain, "example.com", "", "", ban)
	if err != nil {
		t.Fatal(err)
	}
	m := &manifest.Manifest{
		Domain: "example.com",
		Combos: []combo.Spec{
			{Type: combo.TypeVLESSReality, UUID: "uuid-x", Port: 443},
			{Type: combo.TypeHysteria2, Port: 36712, Hy2Password: "pw"},
		},
		Reality: rt,
	}

	links := All(m)
	if len(links) != 2 {
		t.Fatalf("链接数=%d, want 2", len(links))
	}
	for _, l := range links {
		switch l.Combo {
		case combo.TypeVLESSReality:
			if !strings.HasPrefix(l.URL, "vless://uuid-x@example.com:443?") {
				t.Errorf("reality 链接前缀异常: %s", l.URL)
			}
			for _, want := range []string{"security=reality", "pbk=", "flow=xtls-rprx-vision", "sni=example.com"} {
				if !strings.Contains(l.URL, want) {
					t.Errorf("reality 链接缺少 %q: %s", want, l.URL)
				}
			}
		case combo.TypeHysteria2:
			if !strings.HasPrefix(l.URL, "hysteria2://") {
				t.Errorf("hy2 链接前缀异常: %s", l.URL)
			}
			if !strings.Contains(l.URL, "insecure=1") {
				t.Errorf("hy2 自签应含 insecure=1: %s", l.URL)
			}
		}
	}
}

func TestAllSkipsMalformedReality(t *testing.T) {
	m := &manifest.Manifest{
		Domain: "example.com",
		Combos: []combo.Spec{
			{Type: combo.TypeVLESSReality, UUID: "uuid-x", Port: 443},
			{Type: combo.TypeHysteria2, Port: 36712, Hy2Password: "pw"},
		},
	}
	links := All(m)
	if len(links) != 1 || links[0].Combo != combo.TypeHysteria2 {
		t.Fatalf("malformed REALITY 应跳过且保留 hy2，got %+v", links)
	}
}
