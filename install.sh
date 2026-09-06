#!/usr/bin/env bash
# neu-sbox 安装脚本：自动编译 → 校验 sha256 → 安装到 /usr/local/bin/neu-sbox。
# 用法（源码目录内）:  sudo ./install.sh
#
# 目录含 go.mod 且本机有 go 工具链时，先从源码编译，保证装的是当前代码，
# 避免误装过期的预编译二进制；否则回退到已有的预编译二进制 neu-sbox。
set -euo pipefail

STAGE="$(cd "$(dirname "$0")" && pwd)"
BIN_SRC="$STAGE/neu-sbox"
SHA_FILE="$STAGE/neu-sbox.sha256"

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
        make -C "$STAGE" build
    else
        (cd "$STAGE" && \
            CGO_ENABLED=0 GOOS=linux GOTOOLCHAIN=local go build -trimpath -buildvcs=false \
                -tags=netgo,osusergo \
                -ldflags "-s -w -X main.version=$version" \
                -o neu-sbox . && \
            sha256sum neu-sbox > neu-sbox.sha256)
    fi
    restore_ownership "$BIN_SRC" "$SHA_FILE"
}

if [ -f "$STAGE/go.mod" ] && command -v go >/dev/null 2>&1; then
    echo "building neu-sbox from source..."
    build_from_source
elif [ -x "$BIN_SRC" ]; then
    echo "warning: 未从源码编译（缺少 go 工具链或 go.mod），使用已有二进制 $BIN_SRC" >&2
else
    echo "error: 无法编译且未找到预编译二进制 $BIN_SRC" >&2
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
