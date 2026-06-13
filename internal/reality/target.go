package reality

import (
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

// Source 是 REALITY 目标来源类型。
type Source string

const (
	SourceOwnDomain Source = "own_domain" // 自有域名经 Angie（ACME 真证书）
	SourceCurated   Source = "curated"    // 精选外部目标
	SourceCustom    Source = "custom"     // 用户自定义 SNI
)

// Target 描述 REALITY 前置的完整配置。
type Target struct {
	Source      Source   `json:"source"`
	ServerNames []string `json:"server_names"` // 客户端 SNI
	Target      string   `json:"target"`       // 服务端回源地址 host:port
	PrivateKey  string   `json:"private_key"`  // 服务端私钥（base64url）
	PublicKey   string   `json:"public_key"`   // 客户端公钥（base64url）
	ShortIDs    []string `json:"short_ids"`
}

// Resolve 把用户的来源选择解析为经校验的 REALITY Target，并生成密钥/shortID。
func Resolve(src Source, domain, curatedPick, customSNI string, ban *List) (*Target, error) {
	t := &Target{Source: src}
	switch src {
	case SourceOwnDomain:
		// 回源到本机 Angie 的真站点，借用自有域名的真证书 → 最强伪装。
		t.ServerNames = []string{domain}
		t.Target = "127.0.0.1:8443"
	case SourceCurated:
		sni := strings.TrimSpace(curatedPick)
		if !isLikelyDomain(sni) {
			return nil, fmt.Errorf("curated 目标不合法: %q", sni)
		}
		if hit, rule := ban.Match(sni); hit {
			return nil, fmt.Errorf("curated 目标 %s 命中封禁清单（规则 %s）", sni, rule)
		}
		t.ServerNames = []string{sni}
		t.Target = sni + ":443"
	case SourceCustom:
		sni := strings.ToLower(strings.TrimSpace(customSNI))
		if !isLikelyDomain(sni) {
			return nil, fmt.Errorf("自定义 SNI 不合法: %q", sni)
		}
		if hit, rule := ban.Match(sni); hit {
			return nil, fmt.Errorf("自定义 SNI %s 命中封禁清单（规则 %s）。Apple 等域名禁止用作 REALITY 前置", sni, rule)
		}
		t.ServerNames = []string{sni}
		t.Target = sni + ":443"
	default:
		return nil, fmt.Errorf("未知 REALITY 源: %q", src)
	}

	priv, pub, err := GenerateKeypair()
	if err != nil {
		return nil, err
	}
	t.PrivateKey = priv
	t.PublicKey = pub
	t.ShortIDs = GenerateShortIDs(2)
	return t, nil
}

// GenerateKeypair 生成 X25519 密钥对，base64url（无填充）编码，与 `xray x25519` 输出一致。
// 使用 stdlib crypto/ecdh，无需外部依赖。
func GenerateKeypair() (priv, pub string, err error) {
	key, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		return "", "", fmt.Errorf("生成 X25519 密钥: %w", err)
	}
	priv = base64.RawURLEncoding.EncodeToString(key.Bytes())
	pub = base64.RawURLEncoding.EncodeToString(key.PublicKey().Bytes())
	return priv, pub, nil
}

// GenerateShortIDs 生成 n 个随机 8 字节 hex shortID，并前置一个空串（兼容无 shortID 的客户端）。
func GenerateShortIDs(n int) []string {
	ids := []string{""}
	for i := 0; i < n; i++ {
		b := make([]byte, 8)
		if _, err := rand.Read(b); err != nil {
			// rand.Read 在 Linux 上实践中不会失败；保险起见中止。
			panic(errors.New("读取随机数失败: " + err.Error()))
		}
		ids = append(ids, hex.EncodeToString(b))
	}
	return ids
}

// isLikelyDomain 做轻量域名格式校验（非空、有点号、标签合法、无端口/路径）。
func isLikelyDomain(s string) bool {
	if s == "" || len(s) > 253 {
		return false
	}
	if strings.ContainsAny(s, " /:") {
		return false
	}
	if !strings.Contains(s, ".") {
		return false
	}
	for _, label := range strings.Split(s, ".") {
		if label == "" || len(label) > 63 {
			return false
		}
	}
	return true
}
