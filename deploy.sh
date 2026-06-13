#!/usr/bin/env bash
# Reality Deployer —— root 入口。
# 用法：
#   sudo ./deploy.sh                 # 向导，结束后询问是否应用
#   sudo ./deploy.sh wizard          # 仅向导
#   sudo ./deploy.sh apply           # 应用到系统
#   sudo ./deploy.sh uninstall       # 卸载
#   ./deploy.sh status|export        # 只读，无需 root
set -euo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
export REALITY_DEPLOYER_ROOT="$HERE"

# 内联最小日志（避免依赖 lib，使本入口自包含）
_rd_die() { printf '\033[31m✗\033[0m %s\n' "$*" >&2; exit 1; }

# 定位二进制
BIN=""
for c in \
  "$HERE/reality-deployer" \
  "$HERE/dist/reality-deployer" \
  "$HERE/dist/reality-deployer-linux-amd64" \
  "$(command -v reality-deployer 2>/dev/null || true)"; do
  if [ -n "$c" ] && [ -x "$c" ]; then BIN="$c"; break; fi
done
[ -n "$BIN" ] || _rd_die "找不到 reality-deployer 二进制。先运行：make build   （或 make build-all 交叉编译）"

SUBCMD="${1:-wizard}"
case "$SUBCMD" in
  apply|uninstall)
    [ "$(id -u)" -eq 0 ] || _rd_die "$SUBCMD 需要 root，请用 sudo。"
    exec "$BIN" "$SUBCMD" "${@:2}"
    ;;
  status|export)
    exec "$BIN" "$SUBCMD" "${@:2}"
    ;;
  wizard|help|-h|--help)
    ;;
    *)
    echo "用法: $0 [wizard|apply|uninstall|status|export]" >&2
    exit 2
    ;;
esac

# 默认：向导 → 询问是否立即应用
"$BIN" wizard
echo
if [ "$(id -u)" -eq 0 ]; then
  read -r -p "立即应用到系统（安装/配置/启动/签发证书）？[y/N] " ans
  case "$ans" in
    y|Y|yes) exec "$BIN" apply ;;
    *) echo "稍后应用：sudo $0 apply" ;;
  esac
else
  echo "向导已完成配置渲染。应用：sudo $0 apply"
fi
