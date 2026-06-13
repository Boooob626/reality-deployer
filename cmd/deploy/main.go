// reality-deployer CLI 入口。
// 子命令：wizard（默认）/ apply / uninstall / status / export
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"reality-deployer/internal/combo"
	"reality-deployer/internal/link"
	"reality-deployer/internal/manifest"
	"reality-deployer/internal/paths"
	"reality-deployer/internal/wizard"
)

func main() {
	cmd := "wizard"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}

	var err error
	switch cmd {
	case "wizard":
		err = wizard.Run()
	case "apply":
		err = runScript("install.sh")
	case "uninstall":
		err = runScript("uninstall.sh")
	case "status":
		err = status()
	case "export":
		err = export()
	case "-h", "--help", "help":
		usage()
		return
	default:
		fmt.Fprintf(os.Stderr, "未知命令: %s\n\n", cmd)
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "✗ %v\n", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Print(`reality-deployer <command>

  wizard      交互式向导（默认）
  apply       执行 install.sh（安装/配置/启动服务/签发证书）
  uninstall   执行 uninstall.sh（停服务/删配置/回收 ufw）
  status      查看当前部署（读 manifest.json）
  export      重新导出客户端连接链接
`)
}

// runScript 定位并执行 scripts/ 下的脚本，透传 stdin/out/err 与退出码。
func runScript(name string) error {
	p, err := scriptPath(name)
	if err != nil {
		return err
	}
	args := append([]string{p}, os.Args[2:]...)
	c := exec.Command("bash", args...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	if err := c.Run(); err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			os.Exit(ee.ExitCode())
		}
		return err
	}
	return nil
}

// scriptPath 查找脚本：优先 $REALITY_DEPLOYER_ROOT/scripts，其次 cwd/scripts。
func scriptPath(name string) (string, error) {
	candidates := []string{
		filepath.Join(os.Getenv("REALITY_DEPLOYER_ROOT"), "scripts", name),
	}
	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(cwd, "scripts", name))
	}
	for _, c := range candidates {
		if c == "" {
			continue
		}
		if _, err := os.Stat(c); err == nil {
			return c, nil
		}
	}
	return "", fmt.Errorf("找不到 scripts/%s（请通过 deploy.sh 入口运行，或设置 REALITY_DEPLOYER_ROOT）", name)
}

func loadManifest() (*manifest.Manifest, error) {
	if !manifest.Exists(paths.Manifest) {
		return nil, fmt.Errorf("无部署记录 (%s)，请先运行 wizard", paths.Manifest)
	}
	return manifest.Load(paths.Manifest)
}

func status() error {
	m, err := loadManifest()
	if err != nil {
		return err
	}
	fmt.Printf("域名:      %s\n", m.Domain)
	fmt.Printf("公网 IP:   %s\n", m.PublicIP)
	fmt.Printf("ACME 邮箱: %s\n", m.Email)
	fmt.Printf("SSH 端口:  %d\n", m.SSHPort)
	fmt.Printf("路由:      %s", m.Routing.Preset)
	if m.Routing.AdBlock {
		fmt.Print(" +adblock")
	}
	fmt.Println()
	if m.Reality != nil {
		fmt.Printf("REALITY:   source=%s target=%s sni=%v\n", m.Reality.Source, m.Reality.Target, m.Reality.ServerNames)
	}
	fmt.Println("组合:")
	for _, c := range m.Combos {
		switch c.Type {
		case combo.TypeVLESSReality:
			fmt.Printf("  - VLESS+Vision+REALITY @ 443/tcp (uuid %s)\n", c.UUID)
		case combo.TypeVLESSTLS:
			fmt.Printf("  - VLESS+Vision+TLS @ 443/tcp (uuid %s)\n", c.UUID)
		case combo.TypeHysteria2:
			fmt.Printf("  - Hysteria2 @ %d/udp\n", c.Port)
		}
	}
	fmt.Println("\n连接链接:")
	for _, l := range link.All(m) {
		fmt.Printf("  [%s] %s\n", l.Name, l.URL)
	}
	return nil
}

func export() error {
	m, err := loadManifest()
	if err != nil {
		return err
	}
	links := link.All(m)
	if len(links) == 0 {
		fmt.Println("(无组合)")
		return nil
	}
	for _, l := range links {
		fmt.Printf("[%s]\n%s\n", l.Name, l.URL)
		printQR(l.URL)
		fmt.Println()
	}
	if _, err := exec.LookPath("qrencode"); err != nil {
		fmt.Println("# 终端二维码需 qrencode：apt install qrencode")
	}
	return nil
}

// printQR 在终端打印二维码（仅当 qrencode 可用时）。
func printQR(s string) {
	if _, err := exec.LookPath("qrencode"); err != nil {
		return
	}
	c := exec.Command("qrencode", "-t", "ANSIUTF8", "-m", "2", s)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	_ = c.Run()
}
