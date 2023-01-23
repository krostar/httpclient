#!/usr/bin/env sh

set -o errexit
set -o nounset

trap ctrl_c INT
ctrl_c() {
	exit 255
}

gofumpt -extra -w "$@"
gci write --custom-order --section "standard,default,prefix(github.com/krostar/),dot" "$@"
