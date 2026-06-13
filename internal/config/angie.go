package config

import (
	"bytes"
	"strings"
	"text/template"

	"reality-deployer/internal/assets"
	"reality-deployer/internal/manifest"
)

// Angie 渲染 Angie 站点配置（含内置 ACME HTTP-01）。
func Angie(m *manifest.Manifest) (string, error) {
	raw, err := assets.FS.ReadFile(assets.PathAngieTmpl)
	if err != nil {
		return "", err
	}
	t, err := template.New("angie").Parse(string(raw))
	if err != nil {
		return "", err
	}
	data := map[string]string{
		"Domain":   m.Domain,
		"Email":    m.Email,
		"ClientID": ClientID(m.Domain),
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// ClientID 把域名归一化为合法 Angie 标识符，作为 acme_client 名称。
// 例：a.example.com → le_a_example_com
func ClientID(domain string) string {
	r := strings.NewReplacer(".", "_", "-", "_")
	return "le_" + r.Replace(strings.ToLower(domain))
}
