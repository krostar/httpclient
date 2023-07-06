#!/usr/bin/env sh

trap ctrl_c INT
ctrl_c() {
	exit 255
}

set -o errexit
set -o nounset
set -o xtrace

gci write "$@" --custom-order --section "standard" --section "default" --section "Prefix(github.com/krostar/)" --section "dot"
gofumpt -extra -w "$@"
