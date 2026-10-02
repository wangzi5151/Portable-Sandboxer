#!/usr/bin/env sh
set -eu
mkdir -p dist
build() { GOOS="$1" GOARCH="$2" go build -trimpath -ldflags="-s -w" -o "dist/portable-sandboxer-$1-$2$3" .; }
build windows amd64 .exe
build windows arm64 .exe
build linux amd64 ''
build linux arm64 ''
printf '%s\n' 'Cross builds attempted. Fyne desktop builds require each target platform C/GUI toolchain; use native runners and smoke-test artifacts.'
