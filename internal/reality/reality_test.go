package reality

import (
	"encoding/base64"
	"testing"
)

func TestBanlistApple(t *testing.T) {
	l, err := LoadBanlist()
	if err != nil {
		t.Fatalf("LoadBanlist: %v", err)
	}
	cases := []struct {
		domain string
		want   bool
	}{
		{"apple.com", true},
		{"www.apple.com", true},
		{"icloud.com", true},
		{"something.mzstatic.com", true},
		{"www.microsoft.com", false},
		{"google.com", false},
		{"", false},
	}
	for _, c := range cases {
		hit, rule := l.Match(c.domain)
		if hit != c.want {
			t.Errorf("Match(%q) = %v (rule %q), want %v", c.domain, hit, rule, c.want)
		}
	}
}

func TestCuratedExcludesBanned(t *testing.T) {
	c, err := LoadCurated()
	if err != nil {
		t.Fatalf("LoadCurated: %v", err)
	}
	if len(c) == 0 {
		t.Fatal("curated 列表为空")
	}
	ban, _ := LoadBanlist()
	for _, sni := range c {
		if hit, rule := ban.Match(sni); hit {
			t.Errorf("curated 含封禁项 %s（命中 %s）", sni, rule)
		}
	}
}

func TestGenerateKeypair(t *testing.T) {
	priv, pub, err := GenerateKeypair()
	if err != nil {
		t.Fatalf("GenerateKeypair: %v", err)
	}
	pb, err := base64.RawURLEncoding.DecodeString(priv)
	if err != nil || len(pb) != 32 {
		t.Errorf("priv 应为 32 字节 base64url, got len=%d err=%v", len(pb), err)
	}
	pk, err := base64.RawURLEncoding.DecodeString(pub)
	if err != nil || len(pk) != 32 {
		t.Errorf("pub 应为 32 字节 base64url, got len=%d err=%v", len(pk), err)
	}
}

func TestGenerateShortIDs(t *testing.T) {
	ids := GenerateShortIDs(2)
	if len(ids) != 3 || ids[0] != "" {
		t.Errorf("shortIDs 应为 [空, hex, hex], got %v", ids)
	}
	for i, id := range ids {
		if i == 0 {
			continue
		}
		if len(id) != 16 { // 8 字节 hex
			t.Errorf("shortID[%d] 长度异常: %q", i, id)
		}
	}
}

func TestResolveOwnDomain(t *testing.T) {
	ban, _ := LoadBanlist()
	rt, err := Resolve(SourceOwnDomain, "example.com", "", "", ban)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if rt.Target != "127.0.0.1:8443" {
		t.Errorf("own_domain target 应为 127.0.0.1:8443, got %s", rt.Target)
	}
	if len(rt.ServerNames) != 1 || rt.ServerNames[0] != "example.com" {
		t.Errorf("serverNames 异常: %v", rt.ServerNames)
	}
	if rt.PrivateKey == "" || rt.PublicKey == "" {
		t.Error("密钥缺失")
	}
}

func TestResolveRejectsApple(t *testing.T) {
	ban, _ := LoadBanlist()
	if _, err := Resolve(SourceCustom, "example.com", "", "www.apple.com", ban); err == nil {
		t.Fatal("自定义 SNI=www.apple.com 应被拒绝")
	}
	if _, err := Resolve(SourceCurated, "example.com", "www.apple.com", "", ban); err == nil {
		t.Fatal("curated 传入 apple 应被拒绝")
	}
}

func TestResolveRejectsBadDomain(t *testing.T) {
	ban, _ := LoadBanlist()
	if _, err := Resolve(SourceCustom, "example.com", "", "not a domain", ban); err == nil {
		t.Fatal("非法 SNI 应被拒绝")
	}
}
