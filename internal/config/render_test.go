package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"reality-deployer/internal/combo"
	"reality-deployer/internal/manifest"
	"reality-deployer/internal/reality"
)

func mustReality(t *testing.T) *reality.Target {
	t.Helper()
	ban, err := reality.LoadBanlist()
	if err != nil {
		t.Fatal(err)
	}
	rt, err := reality.Resolve(reality.SourceOwnDomain, "example.com", "", "", ban)
	if err != nil {
		t.Fatal(err)
	}
	return rt
}

func TestXrayRenderProducesValidJSON(t *testing.T) {
	rt := mustReality(t)
	m := &manifest.Manifest{
		Domain:  "example.com",
		Email:   "a@example.com",
		SSHPort: 22,
		Combos: []combo.Spec{
			{Type: combo.TypeVLESSReality, UUID: "11111111-1111-1111-1111-111111111111", Port: 443, Flow: "xtls-rprx-vision"},
			{Type: combo.TypeHysteria2, Port: 36712, Hy2Password: "secret"},
		},
		Reality: rt,
		Routing: manifest.Routing{Preset: "block_cn"},
	}
	b, err := Xray(m)
	if err != nil {
		t.Fatalf("Xray: %v", err)
	}
	var cfg map[string]any
	if err := json.Unmarshal(b, &cfg); err != nil {
		t.Fatalf("非合法 JSON: %v\n%s", err, b)
	}

	ins, ok := cfg["inbounds"].([]any)
	if !ok || len(ins) != 2 {
		t.Fatalf("inbounds 数异常: %v", cfg["inbounds"])
	}
	first, _ := ins[0].(map[string]any)
	if first["protocol"] != "vless" {
		t.Errorf("首个 inbound 应为 vless, got %v", first["protocol"])
	}
	ss, _ := first["streamSettings"].(map[string]any)
	if ss["security"] != "reality" {
		t.Errorf("security 应为 reality, got %v", ss["security"])
	}
	rs, _ := ss["realitySettings"].(map[string]any)
	if rs["target"] != "127.0.0.1:8443" {
		t.Errorf("reality target 异常: %v", rs["target"])
	}
	if rs["privateKey"] == "" || rs["privateKey"] == nil {
		t.Error("reality privateKey 缺失")
	}
	sniff, _ := first["sniffing"].(map[string]any)
	if sniff["routeOnly"] != true {
		t.Errorf("sniffing.routeOnly 应启用，got %v", sniff["routeOnly"])
	}

	// routing 含 CN 屏蔽
	if !strings.Contains(string(b), "geosite:cn") {
		t.Error("缺 geosite:cn 路由规则")
	}
	if !strings.Contains(string(b), `"domainStrategy": "AsIs"`) {
		t.Error("routing 应使用 AsIs 避免额外 DNS 解析")
	}
	if !strings.Contains(string(b), `"policy"`) || !strings.Contains(string(b), `"uplinkOnly": 0`) {
		t.Error("缺少隐私/低延迟 policy")
	}
	// outbounds 含 direct + block
	if !strings.Contains(string(b), `"tag": "direct"`) || !strings.Contains(string(b), `"tag": "block"`) {
		t.Error("缺少 direct/block 出站")
	}
}

func TestVLESSXHTTPInboundSchema(t *testing.T) {
	m := &manifest.Manifest{
		Domain: "example.com", Email: "a@example.com", SSHPort: 22,
		Combos: []combo.Spec{
			{Type: combo.TypeVLESSXHTTP, UUID: "uuid-x", Port: 443, XHTTPPath: "/rd-test", XHTTPMode: "auto"},
		},
		Routing: manifest.Routing{Preset: "block_cn_ru"},
	}
	b, err := Xray(m)
	if err != nil {
		t.Fatal(err)
	}
	var cfg map[string]any
	if err := json.Unmarshal(b, &cfg); err != nil {
		t.Fatalf("非合法 JSON: %v", err)
	}
	in := cfg["inbounds"].([]any)[0].(map[string]any)
	ss, _ := in["streamSettings"].(map[string]any)
	if ss["network"] != "xhttp" || ss["security"] != "tls" {
		t.Fatalf("xhttp streamSettings 异常: %+v", ss)
	}
	xs, _ := ss["xhttpSettings"].(map[string]any)
	if xs["path"] != "/rd-test" || xs["mode"] != "auto" {
		t.Fatalf("xhttpSettings 异常: %+v", xs)
	}
	extra, _ := xs["extra"].(map[string]any)
	if extra["scStreamUpServerSecs"] != "20-80" {
		t.Fatalf("xhttp extra 缺少 stream-up 保活: %+v", extra)
	}
}

func TestHysteria2InboundSchema(t *testing.T) {
	m := &manifest.Manifest{
		Domain: "example.com", Email: "a@example.com", SSHPort: 22,
		Combos: []combo.Spec{
			{Type: combo.TypeHysteria2, Port: 36712, Hy2Password: "pw", Hy2Obfs: "salamander", Hy2ObfsPwd: "obfspw"},
		},
		Routing: manifest.Routing{Preset: "none"},
	}
	b, err := Xray(m)
	if err != nil {
		t.Fatal(err)
	}
	var cfg map[string]any
	if err := json.Unmarshal(b, &cfg); err != nil {
		t.Fatalf("非合法 JSON: %v", err)
	}
	in := cfg["inbounds"].([]any)[0].(map[string]any)
	if in["protocol"] != "hysteria" {
		t.Errorf("protocol 应为 hysteria（version=2），got %v", in["protocol"])
	}
	if settings, _ := in["settings"].(map[string]any); settings["version"] != float64(2) {
		t.Errorf("settings.version 应为 2，got %v", settings["version"])
	}
	ss, _ := in["streamSettings"].(map[string]any)
	if ss["network"] != "hysteria" {
		t.Errorf("network 应为 hysteria，got %v", ss["network"])
	}
	fm, ok := ss["finalmask"].(map[string]any)
	if !ok {
		t.Fatal("salamander obfs 应在 streamSettings.finalmask")
	}
	udp := fm["udp"].([]any)[0].(map[string]any)
	if udp["type"] != "salamander" {
		t.Errorf("finalmask.udp[0].type 应为 salamander，got %v", udp["type"])
	}
}

func TestUFWScriptHasApplyRevert(t *testing.T) {
	fw := manifest.Firewall{
		Rules: []manifest.UFWRule{
			{Action: "allow", Proto: "tcp", Port: 22, Note: "ssh"},
			{Action: "allow", Proto: "udp", Range: "20000:50000", Note: "hy2-hop"},
		},
	}
	s := UFWScript(fw)
	for _, want := range []string{"ufw_apply()", "ufw_revert()", "22/tcp", "20000:50000/udp", "|| true"} {
		if !strings.Contains(s, want) {
			t.Errorf("ufw 脚本缺少 %q\n%s", want, s)
		}
	}
}

func TestApplyEnvContainsDomainAndCombos(t *testing.T) {
	rt := mustReality(t)
	m := &manifest.Manifest{
		Domain: "example.com", Email: "a@example.com", SSHPort: 22,
		Combos: []combo.Spec{
			{Type: combo.TypeVLESSReality, UUID: "u", Port: 443},
			{Type: combo.TypeHysteria2, Port: 36712, Hy2Password: "p"},
		},
		Reality: rt,
		Tuning:  manifest.Tuning{KernelLowLatency: true},
	}
	env := ApplyEnv(m, "/tmp/staging")
	for _, want := range []string{"DOMAIN=\"example.com\"", "ENABLE_VLESS_REALITY=1", "ENABLE_VLESS_XHTTP=0", "ENABLE_HYSTERIA2=1", "HY2_PORT=36712", "REALITY_SOURCE=\"own_domain\"", "KERNEL_LOW_LATENCY=1"} {
		if !strings.Contains(env, want) {
			t.Errorf("apply.env 缺少 %q\n%s", want, env)
		}
	}
}

func TestRenderWritesAllArtifacts(t *testing.T) {
	rt := mustReality(t)
	m := &manifest.Manifest{
		Domain: "example.com", Email: "a@example.com", SSHPort: 22,
		Combos: []combo.Spec{
			{Type: combo.TypeVLESSReality, UUID: "u", Port: 443},
			{Type: combo.TypeHysteria2, Port: 36712, Hy2Password: "p"},
		},
		Reality: rt,
		Routing: manifest.Routing{Preset: "block_cn"},
		Firewall: manifest.BuildFirewall([]combo.Spec{
			{Type: combo.TypeVLESSReality, Port: 443},
			{Type: combo.TypeHysteria2, Port: 36712},
		}, 22),
	}
	base := t.TempDir()
	if err := Render(m, base); err != nil {
		t.Fatalf("Render: %v", err)
	}
	for _, rel := range []string{
		"staging/xray/config.json",
		"staging/angie/example.com.conf",
		"staging/decoy/index.html",
		"staging/systemd/xray.service",
		"staging/ufw_rules.sh",
		"apply.env",
	} {
		if _, err := os.Stat(filepath.Join(base, rel)); err != nil {
			t.Errorf("缺少产物 %s: %v", rel, err)
		}
	}
	// 渲染出的 xray 配置应为合法 JSON
	b, err := os.ReadFile(filepath.Join(base, "staging/xray/config.json"))
	if err != nil {
		t.Fatal(err)
	}
	var cfg map[string]any
	if err := json.Unmarshal(b, &cfg); err != nil {
		t.Errorf("staging xray config 非合法 JSON: %v", err)
	}
	// angie conf 含 ACME client 与域名
	angie, _ := os.ReadFile(filepath.Join(base, "staging/angie/example.com.conf"))
	for _, want := range []string{
		"resolver 1.1.1.1 8.8.8.8",
		"acme_client le_example_com https://acme-v02.api.letsencrypt.org/directory",
		"acme le_example_com;",
		"$acme_cert_le_example_com",
		"example.com",
	} {
		if !strings.Contains(string(angie), want) {
			t.Errorf("angie conf 缺少 %q", want)
		}
	}
	if strings.Contains(string(angie), "acme_challenge") || strings.Contains(string(angie), "le_le_") {
		t.Errorf("angie conf 含旧 ACME 写法:\n%s", string(angie))
	}
	if strings.Contains(string(angie), "acme_client_path /etc/angie/acme") {
		t.Errorf("angie conf 不应把 ACME 存储改到 /etc:\n%s", string(angie))
	}
}
