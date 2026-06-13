package link

import (
	"encoding/json"
	"net/url"
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

func TestVLESSXHTTPLink(t *testing.T) {
	m := &manifest.Manifest{
		Domain: "example.com",
		Combos: []combo.Spec{
			{Type: combo.TypeVLESSXHTTP, UUID: "uuid-x", Port: 443, XHTTPPath: "/rd-test", XHTTPMode: "auto"},
		},
	}
	links := All(m)
	if len(links) != 1 {
		t.Fatalf("链接数=%d, want 1", len(links))
	}
	u := links[0].URL
	for _, want := range []string{"vless://uuid-x@example.com:443?", "security=tls", "type=xhttp", "path=%2Frd-test", "mode=auto", "extra="} {
		if !strings.Contains(u, want) {
			t.Fatalf("xhttp link 缺少 %q: %s", want, u)
		}
	}
	parsed, err := url.Parse(u)
	if err != nil {
		t.Fatal(err)
	}
	var extra map[string]any
	if err := json.Unmarshal([]byte(parsed.Query().Get("extra")), &extra); err != nil {
		t.Fatalf("xhttp extra 非 JSON: %v", err)
	}
	xmux, _ := extra["xmux"].(map[string]any)
	if xmux["maxConcurrency"] != "16-32" || xmux["hMaxRequestTimes"] != "600-900" {
		t.Fatalf("xhttp extra 缺少 XMUX 随机范围: %+v", extra)
	}
}
