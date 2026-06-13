package config

import (
	"bytes"
	htmltmpl "html/template"

	"reality-deployer/internal/assets"
)

// Systemd 返回 Xray 的 systemd unit（模板无占位符，原样读取）。
func Systemd() (string, error) {
	b, err := assets.FS.ReadFile(assets.PathUnitTmpl)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// Decoy 渲染伪装站点首页（HTML 模板，域名注入）。
func Decoy(domain string) (string, error) {
	raw, err := assets.FS.ReadFile(assets.PathDecoyHTML)
	if err != nil {
		return "", err
	}
	t, err := htmltmpl.New("decoy").Parse(string(raw))
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, map[string]string{"Domain": domain}); err != nil {
		return "", err
	}
	return buf.String(), nil
}
