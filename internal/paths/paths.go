// Package paths 集中所有目标系统路径，避免字符串散落。
// 与 Xray 官方安装脚本默认路径保持一致。
package paths

const (
	XrayBin     = "/usr/local/bin/xray"
	XrayConfDir = "/usr/local/etc/xray"
	XrayConf    = "/usr/local/etc/xray/config.json"
	XrayLogDir  = "/var/log/xray"

	AngieConfDir = "/etc/angie"
	AngieSiteDir = "/etc/angie/conf.d"
	AngieACMEDir = "/etc/angie/acme"
	AngieACMEKey = "/etc/angie/acme/account.key"

	ManifestDir = "/var/lib/reality-deployer"
	Manifest    = "/var/lib/reality-deployer/manifest.json"
	StagingDir  = "/var/lib/reality-deployer/staging"
	ApplyEnv    = "/var/lib/reality-deployer/apply.env"
)

const systemdXray = "/etc/systemd/system/xray.service"

// 以下函数把 <domain> 代入路径。

func AngieSite(domain string) string  { return AngieSiteDir + "/" + domain + ".conf" }
func AngieWWW(domain string) string   { return "/var/www/" + domain }
func DomainCert(domain string) string { return AngieACMEDir + "/" + domain + ".fullchain.pem" }
func DomainKey(domain string) string  { return AngieACMEDir + "/" + domain + ".key" }

// SystemdXray 是 Xray systemd unit 路径（与 domain 无关）。
func SystemdXray() string { return systemdXray }

// Hysteria2 自签证书（与 domain 无关，固定路径）。
const (
	Hy2Cert = "/usr/local/etc/xray/hy2.crt"
	Hy2Key  = "/usr/local/etc/xray/hy2.key"
)
