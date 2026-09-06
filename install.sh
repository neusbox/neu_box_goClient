#!/usr/bin/env bash
# neu-sbox 安装脚本：自动编译（需 Go >= 1.18）→ 校验 sha256 → 安装到 /usr/local/bin/neu-sbox。
# 用法（源码目录内）:  sudo ./install.sh
#
# 默认行为:
#   - 目录含 go.mod 且有可用 go（>= 1.18）: 先从源码编译，保证装的是当前代码，
#     避免误装过期的预编译二进制。
#   - 无 go.mod（发布包）: 直接安装预编译的 neu-sbox（校验 sha256）。
#
# 环境变量:
#   NEU_SBOX_GO            系统 go 太旧时指向新版 go，如 NEU_SBOX_GO=$HOME/go/bin/go
#   NEU_SBOX_SKIP_BUILD=1  跳过编译，直接安装目录里已有的 neu-sbox
set -euo pipefail

STAGE="$(cd "$(dirname "$0")" && pwd)"
BIN_SRC="$STAGE/neu-sbox"
SHA_FILE="$STAGE/neu-sbox.sha256"
GO_BIN="${NEU_SBOX_GO:-go}"

# go_usable: $GO_BIN 存在且版本 >= 1.18 时返回 0
go_usable() {
    command -v "$GO_BIN" >/dev/null 2>&1 || return 1
    local mm major minor
    mm="$("$GO_BIN" version 2>/dev/null | sed -n 's/.*go\([0-9][0-9]*\)\.\([0-9][0-9]*\).*/\1 \2/p')" || return 1
    [ -n "$mm" ] || return 1
    major="${mm%% *}"
    minor="${mm##* }"
    if [ "$major" -gt 1 ]; then
        return 0
    fi
    [ "$major" -eq 1 ] && [ "$minor" -ge 18 ]
}

# 以 sudo 运行时，把编译产物属主还原给调用者，避免源码目录里留下 root 属主文件。
restore_ownership() {
    if [ -n "${SUDO_UID:-}" ] && [ -n "${SUDO_GID:-}" ]; then
        chown "${SUDO_UID}:${SUDO_GID}" "$@" 2>/dev/null || true
    fi
}

build_from_source() {
    local version
    version="$(tr -d '[:space:]' < "$STAGE/VERSION" 2>/dev/null || echo dev)"
    if command -v make >/dev/null 2>&1; then
        make -C "$STAGE" GOCMD="$GO_BIN" build
    else
        (cd "$STAGE" && \
            CGO_ENABLED=0 GOOS=linux GOTOOLCHAIN=local "$GO_BIN" build -trimpath -buildvcs=false \
                -tags=netgo,osusergo \
                -ldflags "-s -w -X main.version=$version" \
                -o neu-sbox . && \
            sha256sum neu-sbox > neu-sbox.sha256)
    fi
    restore_ownership "$BIN_SRC" "$SHA_FILE"
}

die_need_go() {
    {
        echo "error: 无法从源码编译（本项目需要 Go >= 1.18）"
        if command -v "$GO_BIN" >/dev/null 2>&1; then
            echo "       当前工具链: $("$GO_BIN" version 2>&1 | head -n 1)"
        else
            echo "       PATH 中未找到 '$GO_BIN'"
        fi
        cat <<'EOF'
解决办法:
  1) 在用户目录安装新版 Go（无需 root），再用 NEU_SBOX_GO 指定:
       从 https://go.dev/dl/ 下载 go1.22.x.linux-<arch>.tar.gz（arch 与 uname -m 对应）
       tar -C ~ -xzf go1.22.x.linux-<arch>.tar.gz
       sudo NEU_SBOX_GO=$HOME/go/bin/go ./install.sh
  2) 若目录里已有编译好的 neu-sbox，可跳过编译直接安装:
       sudo NEU_SBOX_SKIP_BUILD=1 ./install.sh
EOF
    } >&2
    exit 1
}

if [ "${NEU_SBOX_SKIP_BUILD:-0}" = "1" ]; then
    [ -x "$BIN_SRC" ] || { echo "error: NEU_SBOX_SKIP_BUILD=1 但未找到 $BIN_SRC" >&2; exit 1; }
    echo "warning: NEU_SBOX_SKIP_BUILD=1，使用已有二进制 $BIN_SRC（跳过编译）" >&2
elif [ -f "$STAGE/go.mod" ] && go_usable; then
    echo "building neu-sbox from source..."
    build_from_source
elif [ -f "$STAGE/go.mod" ]; then
    die_need_go
elif [ -x "$BIN_SRC" ]; then
    echo "warning: 未找到 go.mod（按发布包处理），使用已有二进制 $BIN_SRC" >&2
else
    echo "error: 未找到预编译二进制 $BIN_SRC，也没有可编译的源码" >&2
    exit 1
fi

if [ ! -x "$BIN_SRC" ]; then
    echo "error: 编译后仍未找到可执行的 $BIN_SRC" >&2
    exit 1
fi

if [ -f "$SHA_FILE" ]; then
    (cd "$STAGE" && sha256sum -c "$(basename "$SHA_FILE")")
else
    echo "warning: 未找到 $SHA_FILE，跳过校验" >&2
fi

DEST=/usr/local/bin/neu-sbox
install -m 0755 "$BIN_SRC" "$DEST"
echo "installed: $($DEST version 2>/dev/null || echo "$DEST")"
