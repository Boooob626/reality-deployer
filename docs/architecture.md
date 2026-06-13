# 架构

## 1. 角色划分

```
                ┌───────────────────────────┐
  用户 ───►     │  Go 向导 (reality-deployer) │  交互 + 校验 + 渲染
                └──────────────┬────────────┘
                               │ 写出
                ┌──────────────▼────────────┐
                │ staging/  (渲染后的配置)   │
                │ apply.env (安装参数)       │
                │ manifest.json (声明式记录) │
                └──────────────┬────────────┘
                               │ 执行
                ┌──────────────▼────────────┐
                │  install.sh (静态通用)     │  Bash 系统变更层
                │  lib/ (distro/pkg/ufw/...) │
                └───────────────────────────┘
```

- **Go** 负责一切"决策与生成"：它把用户的回答 + 校验结果，渲染成静态配置文件、`apply.env` 与 `manifest.json`。
- **Bash** 负责一切"系统变更"：apt、ufw、systemctl、文件落位、ACME 触发。`scripts/install.sh` 是通用的（不随用户选项变化），它读取 `apply.env` 并落位 `staging/`。
- 二者的契约是磁盘上的三组产物，没有跨语言 RPC，简单可靠。

## 2. 为什么不让 Go 直接调 systemctl/apt

跨发行版/版本行为差异大、错误处理脆弱；Go 擅长的是结构化渲染与校验，把系统操作交给 shell 更稳。

## 3. 端口与流量拓扑

```
            443/tcp ──► Xray (REALITY / VLESS-TLS / VLESS-XHTTP)
                            │ target / fallback
                            ▼
            127.0.0.1:8443 ──► Angie (真站点 + ACME 真证书)
            80/tcp   ──► Angie (ACME HTTP-01 + 跳转 https)
            <udp>/   ──► Xray Hysteria2（若选）
```

- 443 归 Xray：REALITY、VLESS-TLS、VLESS-XHTTP 三者择一占用；REALITY 的 `own_domain` 源回到 Angie，浏览器访问域名看到带真证书的真站点。
- 80 归 Angie：HTTP-01 挑战 + 跳转。
- Angie 的 ACME 证书同时供 Xray VLESS-TLS / VLESS-XHTTP 组合与自身使用。
- 若启用低延迟调优，安装层会写 `/etc/sysctl.d/99-reality-deployer.conf`，只包含 fq、BBR（内核支持时）与 TCP Fast Open；卸载时回收。

## 4. 幂等与可逆

- `manifest.json` 是唯一事实来源：记录全部组合、UUID、端口、文件、ufw 规则。
- 重跑 `deploy.sh`：向导识别既有 manifest，提供 reconfigure（重算差异）或直接 apply（幂等，不重复开端口）。
- `uninstall.sh`：按 manifest 精确回收 ufw 规则、删配置、停服务；证书目录可选保留。
