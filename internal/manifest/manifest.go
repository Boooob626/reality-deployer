// Package manifest 定义向导产出 → 安装器消费的契约结构，并提供磁盘读写。
package manifest

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"reality-deployer/internal/combo"
	"reality-deployer/internal/reality"
)

// SchemaVersion 是 manifest 的版本标识，便于后续迁移。
const SchemaVersion = "reality-deployer/v1"

// Manifest 是部署的唯一事实来源。
type Manifest struct {
	Schema   string          `json:"schema"`
	Created  time.Time       `json:"created"`
	Updated  time.Time       `json:"updated"`
	Domain   string          `json:"domain"`
	Email    string          `json:"email"`
	PublicIP string          `json:"public_ip"`
	SSHPort  int             `json:"ssh_port"`

	Combos   []combo.Spec    `json:"combos"`
	Reality  *reality.Target `json:"reality,omitempty"`
	Routing  Routing         `json:"routing"`
	Firewall Firewall        `json:"firewall"`
}

// Routing 描述路由预设。
type Routing struct {
	Preset  string `json:"preset"`
	AdBlock bool   `json:"ad_block"`
}

// Firewall 描述 ufw 规则集。
type Firewall struct {
	DefaultInbound string    `json:"default_inbound"`
	Rules          []UFWRule `json:"rules"`
}

// UFWRule 对应一条 ufw 规则。
type UFWRule struct {
	Action string `json:"action"`          // allow
	Proto  string `json:"proto"`           // tcp / udp
	Port   int    `json:"port,omitempty"`  // 单端口
	Range  string `json:"range,omitempty"` // 端口区间 "20000:50000"
	Note   string `json:"note"`
}

// New 创建一个带默认值的空 Manifest。
func New() *Manifest {
	now := time.Now()
	return &Manifest{
		Schema:  SchemaVersion,
		Created: now,
		Updated: now,
		SSHPort: 22,
		Routing: Routing{Preset: "block_cn"},
	}
}

// BuildFirewall 依据所选组合 + SSH 端口计算 ufw 规则。
func BuildFirewall(combos []combo.Spec, sshPort int) Firewall {
	fw := Firewall{DefaultInbound: "deny"}
	fw.Rules = append(fw.Rules,
		UFWRule{Action: "allow", Proto: "tcp", Port: sshPort, Note: "ssh"},
		UFWRule{Action: "allow", Proto: "tcp", Port: 80, Note: "angie/acme-http01"},
	)
	need443 := false
	for _, c := range combos {
		switch c.Type {
		case combo.TypeVLESSReality, combo.TypeVLESSTLS:
			need443 = true
		case combo.TypeHysteria2:
			fw.Rules = append(fw.Rules, UFWRule{Action: "allow", Proto: "udp", Port: c.Port, Note: "hysteria2"})
			if c.Hy2PortHop != "" {
				fw.Rules = append(fw.Rules, UFWRule{Action: "allow", Proto: "udp", Range: c.Hy2PortHop, Note: "hysteria2-port-hop"})
			}
		}
	}
	if need443 {
		fw.Rules = append(fw.Rules, UFWRule{Action: "allow", Proto: "tcp", Port: 443, Note: "vless"})
	}
	return fw
}

// Save 把 manifest 写入磁盘（创建父目录）。
func (m *Manifest) Save(path string) error {
	m.Updated = time.Now()
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

// Load 从磁盘读取 manifest。
func Load(path string) (*Manifest, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m Manifest
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("解析 manifest %s: %w", path, err)
	}
	return &m, nil
}

// Exists 报告 manifest 文件是否存在（用于检测既有部署）。
func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
