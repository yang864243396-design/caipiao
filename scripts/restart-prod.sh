#!/usr/bin/env bash
# 线上快速更新并重启：后端 systemd + 前端静态构建（nginx 读 dist，一般无需 reload）
#
# 用法：
#   ./scripts/restart-prod.sh              # 拉代码 + 构建前后端 + 重启后端
#   ./scripts/restart-prod.sh --backend    # 仅后端
#   ./scripts/restart-prod.sh --frontend   # 仅 client + admin
#   ./scripts/restart-prod.sh --client     # 仅用户端
#   ./scripts/restart-prod.sh --admin      # 仅管理端
#   ./scripts/restart-prod.sh --no-pull    # 不 git pull
#   ./scripts/restart-prod.sh --migrate    # 构建前执行数据库迁移（默认开启）
#   ./scripts/restart-prod.sh --no-migrate # 跳过数据库迁移
#   ./scripts/restart-prod.sh --reload-nginx
#
# 首次使用：按实际环境改下方「配置区」。

set -euo pipefail

# ========== 配置区（按线上实际修改）==========
APP_ROOT="${APP_ROOT:-/opt/caipiao}"
BACKEND_UNIT="${BACKEND_UNIT:-caipiao-backend}"
BACKEND_DIR="${BACKEND_DIR:-$APP_ROOT/backend}"
CLIENT_DIR="${CLIENT_DIR:-$APP_ROOT/client}"
ADMIN_DIR="${ADMIN_DIR:-$APP_ROOT/admin}"
SERVER_BIN="${SERVER_BIN:-$BACKEND_DIR/bin/server}"
GO_BIN="${GO_BIN:-go}"
NPM_BIN="${NPM_BIN:-npm}"
# 是否在 npm build 前执行 npm ci（依赖有变更时建议 1）
NPM_CI="${NPM_CI:-1}"
# ============================================

DO_PULL=1
DO_BACKEND=1
DO_CLIENT=1
DO_ADMIN=1
# 默认随后端更新执行迁移，避免模拟配额等新列缺失导致 start 500
DO_MIGRATE=1
DO_RELOAD_NGINX=0
SCOPE_SET=0

log()  { printf '\n==> %s\n' "$*"; }
ok()   { printf '    OK: %s\n' "$*"; }
die()  { printf 'ERROR: %s\n' "$*" >&2; exit 1; }

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || die "未找到命令：$1"
}

usage() {
  sed -n '2,16p' "$0" | sed 's/^# \?//'
  exit 0
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    -h|--help) usage ;;
    --no-pull) DO_PULL=0 ;;
    --migrate) DO_MIGRATE=1 ;;
    --no-migrate) DO_MIGRATE=0 ;;
    --reload-nginx) DO_RELOAD_NGINX=1 ;;
    --backend)
      DO_BACKEND=1; DO_CLIENT=0; DO_ADMIN=0; SCOPE_SET=1
      ;;
    --frontend)
      DO_BACKEND=0; DO_CLIENT=1; DO_ADMIN=1; DO_MIGRATE=0; SCOPE_SET=1
      ;;
    --client)
      [[ "$SCOPE_SET" -eq 0 ]] && DO_BACKEND=0 && DO_ADMIN=0 && DO_MIGRATE=0
      DO_CLIENT=1; SCOPE_SET=1
      ;;
    --admin)
      [[ "$SCOPE_SET" -eq 0 ]] && DO_BACKEND=0 && DO_CLIENT=0 && DO_MIGRATE=0
      DO_ADMIN=1; SCOPE_SET=1
      ;;
    *)
      die "未知参数：$1（用 --help 查看用法）"
      ;;
  esac
  shift
done

[[ -d "$APP_ROOT" ]] || die "APP_ROOT 不存在：$APP_ROOT（请改脚本配置区或 export APP_ROOT=...）"

need_cmd systemctl
[[ "$DO_BACKEND" -eq 1 ]] && need_cmd "$GO_BIN"
[[ "$DO_CLIENT" -eq 1 || "$DO_ADMIN" -eq 1 ]] && need_cmd "$NPM_BIN"

if [[ "$DO_PULL" -eq 1 ]]; then
  log "git pull ($APP_ROOT)"
  # 线上目录以远程为准：上次 npm build 常会改脏 components.d.ts，直接 pull 会失败
  if ! git -C "$APP_ROOT" diff --quiet || ! git -C "$APP_ROOT" diff --cached --quiet; then
    log "丢弃本地未提交改动（含构建生成的 *.d.ts），再拉取"
    git -C "$APP_ROOT" reset --hard HEAD
  fi
  git -C "$APP_ROOT" pull --ff-only
  ok "代码已更新"
fi

if [[ "$DO_MIGRATE" -eq 1 ]]; then
  log "数据库迁移"
  [[ -d "$BACKEND_DIR" ]] || die "BACKEND_DIR 不存在：$BACKEND_DIR"
  (
    cd "$BACKEND_DIR"
    "$GO_BIN" run ./cmd/migrate up
  )
  ok "migrate up 完成"
fi

if [[ "$DO_BACKEND" -eq 1 ]]; then
  log "构建后端 → $SERVER_BIN"
  [[ -d "$BACKEND_DIR" ]] || die "BACKEND_DIR 不存在：$BACKEND_DIR"
  mkdir -p "$(dirname "$SERVER_BIN")"
  (
    cd "$BACKEND_DIR"
    "$GO_BIN" build -o "$SERVER_BIN" ./cmd/server
  )
  ok "后端二进制已生成"

  log "重启 systemd: $BACKEND_UNIT"
  sudo systemctl restart "$BACKEND_UNIT"
  sleep 1
  if sudo systemctl is-active --quiet "$BACKEND_UNIT"; then
    ok "$BACKEND_UNIT 运行中"
  else
    sudo systemctl status "$BACKEND_UNIT" --no-pager -l || true
    die "$BACKEND_UNIT 未处于 active，请检查 journalctl -u $BACKEND_UNIT -n 80"
  fi
fi

build_frontend() {
  local name="$1"
  local dir="$2"
  log "构建 $name ($dir)"
  [[ -d "$dir" ]] || die "$name 目录不存在：$dir"
  (
    cd "$dir"
    if [[ "$NPM_CI" -eq 1 ]]; then
      "$NPM_BIN" ci
    fi
    "$NPM_BIN" run build
  )
  ok "$name 构建完成"
}

[[ "$DO_CLIENT" -eq 1 ]] && build_frontend "client" "$CLIENT_DIR"
[[ "$DO_ADMIN"  -eq 1 ]] && build_frontend "admin"  "$ADMIN_DIR"

if [[ "$DO_RELOAD_NGINX" -eq 1 ]]; then
  log "reload nginx"
  need_cmd nginx
  sudo nginx -t
  sudo systemctl reload nginx
  ok "nginx 已 reload"
fi

log "全部完成"
[[ "$DO_BACKEND" -eq 1 ]] && printf '  后端: systemctl status %s\n' "$BACKEND_UNIT"
[[ "$DO_CLIENT" -eq 1 ]] && printf '  用户端: %s/dist\n' "$CLIENT_DIR"
[[ "$DO_ADMIN"  -eq 1 ]] && printf '  管理端: %s/dist\n' "$ADMIN_DIR"
