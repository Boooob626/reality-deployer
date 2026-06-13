package wizard

import (
	"fmt"
	"strconv"
	"strings"

	"reality-deployer/internal/combo"
	"reality-deployer/internal/config"
	"reality-deployer/internal/link"
	"reality-deployer/internal/manifest"
	"reality-deployer/internal/paths"
	"reality-deployer/internal/reality"
	"reality-deployer/internal/routing"
)

const banner = `
┌─────────────────────────────────────────────┐
│        Reality Deployer  —  向导             │
│ VLESS REALITY/TLS/XHTTP · Hysteria2         │
│   Angie (内置 ACME) · ufw · systemd         │
└─────────────────────────────────────────────┘`

// pf 是 reconfigure 时的预填来源（既有 manifest）；新部署时为 nil。
// 向导是单线程交互，用包级变量避免给每个 ask 函数加参数。
var pf *manifest.Manifest

// Run 执行完整交互式向导：收集 → 校验 → 渲染 → 持久化 manifest。
func Run() error {
	fmt.Println(banner)
	fmt.Println("\n本向导只渲染 staging/apply.env/manifest；实际改系统由 install.sh 完成。")

	pf = nil
	if manifest.Exists(paths.Manifest) {
		if existing, err := manifest.Load(paths.Manifest); err == nil {
			pf = existing
		}
		fmt.Printf("\n⚠  检测到既有部署：%s\n", paths.Manifest)
		fmt.Println("   进入 reconfigure 模式：各提示的默认值取自现有配置（回车=保留，输入=修改）。")
		if !confirm("继续？", true) {
			return nil
		}
	}

	ban, err := reality.LoadBanlist()
	if err != nil {
		return fmt.Errorf("加载 REALITY 封禁清单: %w", err)
	}
	curated, _ := reality.LoadCurated()

	m := manifest.New()

	// 1. 域名 + DNS 校验
	if err := askDomain(m); err != nil {
		return err
	}
	// 2. 邮箱
	if err := askEmail(m); err != nil {
		return err
	}
	// 3. 协议组合
	hasReality, hasTLS, hasXHTTP, hasHy2 := askCombos()
	// 443 仲裁
	if count443(hasReality, hasTLS, hasXHTTP) > 1 {
		fmt.Println("\n⚠  VLESS-REALITY / VLESS-TLS / VLESS-XHTTP 都占用 443/tcp，不能并存。")
		keep := selectOpt("保留哪一个占 443？", []opt{
			{"reality", "VLESS + Vision + REALITY（推荐）"},
			{"tls", "VLESS + Vision + TLS（自有证书）"},
			{"xhttp", "VLESS + XHTTP + TLS（较新客户端，XMUX-capable）"},
		}, 0)
		hasReality = keep == "reality"
		hasTLS = keep == "tls"
		hasXHTTP = keep == "xhttp"
	}
	if hasXHTTP {
		if !confirm("启用 XHTTP 需要较新的 Xray 客户端。确认继续？", true) {
			hasXHTTP = false
			if !hasReality && !hasTLS && !hasHy2 {
				hasReality = true
			}
		}
	}
	// 4. 构造组合 specs（reconfigure 时尽量沿用旧 UUID/密钥）
	var specs []combo.Spec
	if hasReality {
		specs = append(specs, combo.Spec{
			Type: combo.TypeVLESSReality, UUID: keepUUID(combo.TypeVLESSReality), Port: 443, Flow: "xtls-rprx-vision",
		})
	}
	if hasTLS {
		specs = append(specs, combo.Spec{
			Type: combo.TypeVLESSTLS, UUID: keepUUID(combo.TypeVLESSTLS), Port: 443, Flow: "xtls-rprx-vision",
		})
	}
	if hasXHTTP {
		specs = append(specs, askXHTTP())
	}
	if hasHy2 {
		specs = append(specs, askHysteria2())
	}
	// 5. REALITY 目标源
	if hasReality {
		rt, err := askRealityTarget(m.Domain, curated, ban)
		if err != nil {
			return err
		}
		m.Reality = rt
	}
	m.Combos = specs
	// 6. 路由预设
	m.Routing.Preset = selectOpt("\n路由预设", []opt{
		{string(routing.PresetBlockCNRU), "屏蔽 CN + RU 目标（默认，防泄露/投诉）"},
		{string(routing.PresetBlockCN), "仅屏蔽 CN 目标"},
		{string(routing.PresetBypassCNRU), "CN + RU 目标走 VPS direct 出口"},
		{string(routing.PresetBypassCN), "仅 CN 目标走 VPS direct 出口"},
		{string(routing.PresetNone), "不附加区域规则"},
	}, routingDefaultIdx())
	m.Routing.AdBlock = confirm("屏蔽广告域名 (geosite:category-ads-all)？", adblockDefault())
	m.Tuning.KernelLowLatency = confirm("启用低延迟内核调优（fq/BBR/TFO，VPS 全局）？", tuningDefault())
	// 7. SSH 端口
	m.SSHPort = promptPort("SSH 端口（ufw 放行）", sshDefault())
	// 8. 防火墙
	m.Firewall = manifest.BuildFirewall(specs, m.SSHPort)
	// 9. 摘要 + 确认
	printSummary(m)
	if !confirm("\n确认渲染配置（此步不改系统，install.sh 才改）？", true) {
		fmt.Println("已取消，未做任何改动。")
		return nil
	}
	// 10. 渲染 + 持久化
	if err := config.Render(m, paths.ManifestDir); err != nil {
		return fmt.Errorf("渲染配置: %w", err)
	}
	if err := m.Save(paths.Manifest); err != nil {
		return fmt.Errorf("写 manifest: %w", err)
	}
	printResult(m)
	return nil
}

// keepUUID 在 reconfigure 时沿用同类型的旧 UUID，避免客户端要重配。
func keepUUID(t string) string {
	if pf == nil {
		return genUUID()
	}
	for _, c := range pf.Combos {
		if c.Type == t && c.UUID != "" {
			return c.UUID
		}
	}
	return genUUID()
}

func domainDefault() string {
	if pf != nil && pf.Domain != "" {
		return pf.Domain
	}
	return ""
}
func emailDefault(fallback string) string {
	if pf != nil && pf.Email != "" {
		return pf.Email
	}
	return fallback
}
func sshDefault() int {
	if pf != nil && pf.SSHPort != 0 {
		return pf.SSHPort
	}
	return 22
}
func routingDefaultIdx() int {
	if pf != nil {
		switch pf.Routing.Preset {
		case string(routing.PresetBlockCN):
			return 1
		case string(routing.PresetBypassCNRU):
			return 2
		case string(routing.PresetBypassCN):
			return 3
		case string(routing.PresetNone):
			return 4
		}
	}
	return 0
}
func adblockDefault() bool {
	if pf != nil {
		return pf.Routing.AdBlock
	}
	return false
}
func tuningDefault() bool {
	if pf != nil {
		return pf.Tuning.KernelLowLatency
	}
	return true
}

// comboDefaults 返回多选默认下标（来自既有 manifest）。
func comboDefaults() []int {
	if pf == nil {
		return []int{0}
	}
	var d []int
	for _, c := range pf.Combos {
		switch c.Type {
		case combo.TypeVLESSReality:
			d = append(d, 0)
		case combo.TypeVLESSTLS:
			d = append(d, 1)
		case combo.TypeVLESSXHTTP:
			d = append(d, 2)
		case combo.TypeHysteria2:
			d = append(d, 3)
		}
	}
	if len(d) == 0 {
		d = []int{0}
	}
	return d
}

func findPrevHy2() *combo.Spec {
	return findPrev(combo.TypeHysteria2)
}

func findPrev(t string) *combo.Spec {
	if pf == nil {
		return nil
	}
	for i := range pf.Combos {
		if pf.Combos[i].Type == t {
			return &pf.Combos[i]
		}
	}
	return nil
}

func askDomain(m *manifest.Manifest) error {
	for {
		d := strings.ToLower(promptDefault("域名（A 记录需已指向本机）", domainDefault()))
		if d == "" {
			fmt.Println("  域名必填。")
			continue
		}
		if !validDomain(d) {
			fmt.Println("  域名格式无效，请输入普通域名（如 example.com）。")
			continue
		}
		m.Domain = d
		detected := detectPublicIP()
		m.PublicIP = detected
		ips, err := resolveDomain(m.Domain)
		switch {
		case err != nil || len(ips) == 0:
			fmt.Println("  ⚠  无法解析域名 DNS，请先确认 A 记录已生效。")
			if !confirm("  仍要继续？", false) {
				continue
			}
		case detected != "" && !contains(ips, detected):
			fmt.Printf("  ⚠  DNS 解析 %v 与本机检测 IP %s 不一致。\n", ips, detected)
			if !confirm("  仍要继续？", false) {
				continue
			}
		default:
			fmt.Printf("  ✓ 域名解析 %v，本机公网 IP %s\n", ips, detected)
		}
		return nil
	}
}

func askEmail(m *manifest.Manifest) error {
	def := emailDefault("admin@" + m.Domain)
	for {
		e := promptDefault("ACME 邮箱（Let's Encrypt）", def)
		if validEmail(e) {
			m.Email = e
			return nil
		}
		fmt.Println("  邮箱格式无效，重试。")
	}
}

func askCombos() (reality, tls, xhttp, hy2 bool) {
	idx := multiSelect("选择协议组合（可多选）", []opt{
		{"reality", "VLESS + Vision + REALITY / TCP  [主力，推荐]"},
		{"tls", "VLESS + Vision + TLS（自有 ACME 证书）"},
		{"xhttp", "VLESS + XHTTP + TLS（较新客户端，XMUX-capable）"},
		{"hysteria2", "Hysteria2（QUIC/UDP，备选）"},
	}, comboDefaults())
	for _, i := range idx {
		switch i {
		case 0:
			reality = true
		case 1:
			tls = true
		case 2:
			xhttp = true
		case 3:
			hy2 = true
		}
	}
	if !reality && !tls && !xhttp && !hy2 {
		fmt.Println("  未选任何组合，默认选 REALITY。")
		reality = true
	}
	return
}

func count443(values ...bool) int {
	var n int
	for _, v := range values {
		if v {
			n++
		}
	}
	return n
}

func askXHTTP() combo.Spec {
	prev := findPrev(combo.TypeVLESSXHTTP)
	path := "/rd-" + genPathToken(8)
	mode := "auto"
	if prev != nil {
		if prev.XHTTPPath != "" {
			path = prev.XHTTPPath
		}
		if prev.XHTTPMode != "" {
			mode = prev.XHTTPMode
		}
	}
	for {
		path = promptDefault("XHTTP path（必须以 / 开头）", path)
		if strings.HasPrefix(path, "/") && !strings.ContainsAny(path, " \t?#") {
			break
		}
		fmt.Println("  path 必须以 / 开头，且不能含空白、? 或 #。")
	}
	mode = selectOpt("XHTTP mode", []opt{
		{"auto", "auto（默认，客户端/服务端自动选择）"},
		{"stream-up", "stream-up（H2/反代兼容性好）"},
		{"packet-up", "packet-up（HTTP 中间盒兼容性强）"},
		{"stream-one", "stream-one（单请求双向流）"},
	}, xhttpModeDefaultIdx(mode))
	return combo.Spec{
		Type:      combo.TypeVLESSXHTTP,
		UUID:      keepUUID(combo.TypeVLESSXHTTP),
		Port:      443,
		XHTTPPath: path,
		XHTTPMode: mode,
	}
}

func xhttpModeDefaultIdx(mode string) int {
	switch mode {
	case "stream-up":
		return 1
	case "packet-up":
		return 2
	case "stream-one":
		return 3
	default:
		return 0
	}
}

func askHysteria2() combo.Spec {
	prev := findPrevHy2()
	defPort := 36712
	if prev != nil && prev.Port != 0 {
		defPort = prev.Port
	}
	port := promptPort("Hysteria2 UDP 端口", defPort)
	if !portFreeUDP(port) {
		fmt.Printf("  ⚠  UDP %d 可能已被占用；如果这是当前 xray，可继续沿用。\n", port)
	}
	password := genPassword(16)
	if prev != nil && prev.Hy2Password != "" {
		password = prev.Hy2Password
	}
	s := combo.Spec{
		Type:        combo.TypeHysteria2,
		Port:        port,
		Hy2Password: password,
	}
	defObfs := false
	if prev != nil && prev.Hy2Obfs == "salamander" {
		s.Hy2Obfs = "salamander"
		s.Hy2ObfsPwd = prev.Hy2ObfsPwd
		defObfs = true
	}
	if confirm("启用 Salamander obfs（混淆）？", defObfs) {
		s.Hy2Obfs = "salamander"
		if s.Hy2ObfsPwd == "" {
			s.Hy2ObfsPwd = genPassword(16)
		}
	} else {
		s.Hy2Obfs = ""
		s.Hy2ObfsPwd = ""
	}
	defHop := false
	if prev != nil && prev.Hy2PortHop != "" {
		s.Hy2PortHop = prev.Hy2PortHop
		defHop = true
	}
	if confirm("启用端口跳跃（udpHop，需 ufw + iptables 放行区间）？", defHop) {
		def := "20000:50000"
		if s.Hy2PortHop != "" {
			def = s.Hy2PortHop
		}
		for {
			r := promptDefault("端口区间（起:止，如 20000:50000）", def)
			if normalized, ok := normalizePortRange(r); ok {
				s.Hy2PortHop = normalized
				break
			}
			fmt.Println("  端口区间无效，请输入 1-65535 内的 start:end，且 start <= end。")
		}
	} else {
		s.Hy2PortHop = ""
	}
	return s
}

// askRealityTarget 处理 3 种 REALITY 来源选择 + Apple 等封禁校验。
func askRealityTarget(domain string, curated []string, ban *reality.List) (*reality.Target, error) {
	defIdx := 0
	if pf != nil && pf.Reality != nil {
		switch pf.Reality.Source {
		case reality.SourceCurated:
			defIdx = 1
		case reality.SourceCustom:
			defIdx = 2
		}
	}
	srcKey := selectOpt("\nREALITY 目标源（dest/serverNames）", []opt{
		{string(reality.SourceOwnDomain), "自有域名经 Angie（ACME 真证书，最强伪装，推荐）"},
		{string(reality.SourceCurated), "精选外部目标（已剔除 Apple 等）"},
		{string(reality.SourceCustom), "自定义 SNI（实时校验封禁清单）"},
	}, defIdx)

	var curatedPick, customSNI string
	switch reality.Source(srcKey) {
	case reality.SourceCurated:
		opts := make([]opt, len(curated))
		for i, c := range curated {
			opts[i] = opt{Key: c, Desc: c}
		}
		if len(opts) == 0 {
			fmt.Println("  curated 列表为空，回退到自定义。")
			srcKey = string(reality.SourceCustom)
		} else {
			curatedPick = selectOpt("选择目标站点", opts, 0)
		}
	case reality.SourceCustom:
		def := ""
		if pf != nil && pf.Reality != nil && len(pf.Reality.ServerNames) > 0 && pf.Reality.Source == reality.SourceCustom {
			def = pf.Reality.ServerNames[0]
		}
		for {
			customSNI = strings.ToLower(promptDefault("自定义 SNI（如 www.microsoft.com）", def))
			if hit, rule := ban.Match(customSNI); hit {
				fmt.Printf("  ✗ %s 命中封禁清单（规则 %s）。Apple 等域名禁止用作 REALITY 前置，请换一个。\n", customSNI, rule)
				continue
			}
			break
		}
	}
	rt, err := reality.Resolve(reality.Source(srcKey), domain, curatedPick, customSNI, ban)
	if err != nil {
		return nil, err
	}
	carryRealitySecrets(rt)
	return rt, nil
}

func carryRealitySecrets(rt *reality.Target) {
	if pf == nil || pf.Reality == nil || rt == nil {
		return
	}
	if pf.Reality.PrivateKey == "" || pf.Reality.PublicKey == "" || len(pf.Reality.ShortIDs) == 0 {
		return
	}
	rt.PrivateKey = pf.Reality.PrivateKey
	rt.PublicKey = pf.Reality.PublicKey
	rt.ShortIDs = append([]string(nil), pf.Reality.ShortIDs...)
}

func promptPort(label string, def int) int {
	for {
		v := prompt(fmt.Sprintf("%s [%d]", label, def))
		if v == "" {
			if def >= 1 && def <= 65535 {
				return def
			}
			fmt.Println("  默认端口无效，请输入 1-65535 范围内的端口。")
			continue
		}
		p, err := strconv.Atoi(v)
		if err != nil {
			fmt.Println("  请输入数字端口。")
			continue
		}
		if p >= 1 && p <= 65535 {
			return p
		}
		fmt.Println("  端口必须在 1-65535 范围内。")
	}
}

func normalizePortRange(s string) (string, bool) {
	parts := strings.SplitN(strings.TrimSpace(s), ":", 2)
	if len(parts) != 2 {
		return "", false
	}
	start, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
	end, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err1 != nil || err2 != nil || start < 1 || end < 1 || start > 65535 || end > 65535 || start > end {
		return "", false
	}
	return fmt.Sprintf("%d:%d", start, end), true
}

func printSummary(m *manifest.Manifest) {
	fmt.Println("\n──── 部署摘要 ────")
	fmt.Printf("域名:       %s\n", m.Domain)
	fmt.Printf("公网 IP:    %s\n", m.PublicIP)
	fmt.Printf("ACME 邮箱:  %s\n", m.Email)
	fmt.Println("协议组合:")
	for _, c := range m.Combos {
		switch c.Type {
		case combo.TypeVLESSReality:
			fmt.Printf("  - VLESS + Vision + REALITY / TCP @ 443 (UUID %s)\n", c.UUID)
		case combo.TypeVLESSTLS:
			fmt.Printf("  - VLESS + Vision + TLS @ 443 (UUID %s)\n", c.UUID)
		case combo.TypeVLESSXHTTP:
			fmt.Printf("  - VLESS + XHTTP + TLS @ 443 (UUID %s, path %s, mode %s)\n", c.UUID, c.XHTTPPath, c.XHTTPMode)
		case combo.TypeHysteria2:
			extra := ""
			if c.Hy2Obfs != "" {
				extra += " +obfs"
			}
			if c.Hy2PortHop != "" {
				extra += " +porthop(" + c.Hy2PortHop + ")"
			}
			fmt.Printf("  - Hysteria2 @ %d/udp%s\n", c.Port, extra)
		}
	}
	if m.Reality != nil {
		fmt.Printf("REALITY 源:  %s（target=%s）\n", m.Reality.Source, m.Reality.Target)
	}
	fmt.Printf("路由预设:    %s", m.Routing.Preset)
	if m.Routing.AdBlock {
		fmt.Print(" +adblock")
	}
	fmt.Println()
	if m.Tuning.KernelLowLatency {
		fmt.Println("内核调优:    fq/BBR/TFO")
	}
	fmt.Printf("SSH 端口:    %d\n", m.SSHPort)
	fmt.Println("ufw 规则:")
	for _, r := range m.Firewall.Rules {
		spec := r.Range
		if spec == "" {
			spec = fmt.Sprintf("%d/%s", r.Port, r.Proto)
		} else {
			spec += "/" + r.Proto
		}
		fmt.Printf("  - allow %s  (%s)\n", spec, r.Note)
	}
}

func printResult(m *manifest.Manifest) {
	fmt.Println("\n✓ 配置已渲染：")
	fmt.Printf("  staging:    %s/staging/\n", paths.ManifestDir)
	fmt.Printf("  apply.env:  %s/apply.env\n", paths.ManifestDir)
	fmt.Printf("  manifest:   %s\n", paths.Manifest)

	fmt.Println("\n──── 客户端连接链接 ────")
	for _, l := range link.All(m) {
		fmt.Printf("\n[%s]\n  %s\n", l.Name, l.URL)
	}

	fmt.Println("\n──── 下一步 ────")
	fmt.Println("  应用到系统（安装 xray/angie、ufw、iptables 端口跳跃、启动服务、签发证书）：")
	fmt.Println("    sudo ./deploy.sh apply")
	fmt.Println("    # 或直接：sudo bash scripts/install.sh")
	fmt.Println("  验证：")
	fmt.Printf("    xray run -test -c %s\n", paths.XrayConf)
	fmt.Println("    angie -t")
	fmt.Println("    ufw status verbose")
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}
