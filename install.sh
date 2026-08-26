#!/usr/bin/env bash
# neu-sbox 安装脚本：校验 sha256 → 安装到 /usr/local/bin/neu-sbox。
# 用法（解包目录内）:  sudo ./install.sh
set -euo pipefail

STAGE="$(cd "$(dirname "$0")" && pwd)"
BIN_SRC="$STAGE/neu-sbox"
SHA_FILE="$STAGE/neu-sbox.sha256"

if [ ! -x "$BIN_SRC" ]; then
    echo "error: 未找到 $BIN_SRC" >&2
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
