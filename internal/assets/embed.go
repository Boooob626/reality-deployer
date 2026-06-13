// Package assets 嵌入所有静态资源：模板、decoy 站点、REALITY 数据清单。
// 唯一事实来源；向导与渲染器都从这里读取。
package assets

import "embed"

//go:embed files/*
var FS embed.FS

// 资源在 embed.FS 内的相对路径，集中在此便于引用。
const (
	PathBanlist   = "files/reality-banlist.txt"
	PathCurated   = "files/reality-curated.txt"
	PathAngieTmpl = "files/angie-site.conf.tmpl"
	PathUnitTmpl  = "files/xray.service.tmpl"
	PathDecoyHTML = "files/decoy/index.html"
)
