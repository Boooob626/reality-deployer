// Package reality 负责 REALITY 目标源的解析、封禁校验与密钥生成。
package reality

import (
	"bufio"
	"bytes"
	"strings"

	"reality-deployer/internal/assets"
)

// List 是解析后的域名清单（精确 + 通配两条规则集）。
type List struct {
	exact     map[string]struct{}
	wildcards []string // "*.apple.com" 归一化为后缀 "apple.com"
}

// LoadBanlist 读取并解析封禁清单。
func LoadBanlist() (*List, error) { return loadList(assets.PathBanlist) }

func loadList(path string) (*List, error) {
	b, err := assets.FS.ReadFile(path)
	if err != nil {
		return nil, err
	}
	l := &List{exact: make(map[string]struct{})}
	sc := bufio.NewScanner(bytes.NewReader(b))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.ToLower(line)
		if strings.HasPrefix(line, "*.") {
			l.wildcards = append(l.wildcards, strings.TrimPrefix(line, "*."))
		} else {
			l.exact[line] = struct{}{}
		}
	}
	return l, sc.Err()
}

// Match 判断 domain 是否命中清单，命中则返回触发规则用于提示。
func (l *List) Match(domain string) (hit bool, rule string) {
	d := normalizeDomain(domain)
	if d == "" {
		return false, ""
	}
	if _, ok := l.exact[d]; ok {
		return true, d
	}
	for _, suf := range l.wildcards {
		if d == suf || strings.HasSuffix(d, "."+suf) {
			return true, "*." + suf
		}
	}
	return false, ""
}

// LoadCurated 读取精选 REALITY 目标，保持文件顺序并去重，返回纯域名切片。
func LoadCurated() ([]string, error) {
	b, err := assets.FS.ReadFile(assets.PathCurated)
	if err != nil {
		return nil, err
	}
	var out []string
	seen := make(map[string]struct{})
	sc := bufio.NewScanner(bytes.NewReader(b))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// 允许行内 "# 注释"
		if i := strings.Index(line, "#"); i >= 0 {
			line = strings.TrimSpace(line[:i])
		}
		line = strings.ToLower(line)
		if line == "" {
			continue
		}
		if _, dup := seen[line]; dup {
			continue
		}
		seen[line] = struct{}{}
		out = append(out, line)
	}
	return out, sc.Err()
}

func normalizeDomain(s string) string {
	d := strings.ToLower(strings.TrimSpace(s))
	d = strings.TrimSuffix(d, ".")
	return d
}
