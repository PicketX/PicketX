#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
# Build-time tools only. The resulting binaries need no shared libraries/tools.
set -euo pipefail
if [[ $# != 1 || "$1" != /* || -e "$1" || -L "$1" ]]; then
  echo 'Usage: build-controller-bundle.sh NEW_ABSOLUTE_DIRECTORY (must not exist)' >&2
  exit 2
fi
output=$1
cd "$(dirname "$0")/.."
if [[ -n $(git status --porcelain --untracked-files=all) ]]; then
  echo 'Commit source changes before making an attributable bundle' >&2
  exit 2
fi
mkdir -m 755 "$output"
mkdir "$output/bin" "$output/deploy"
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -trimpath -tags 'netgo osusergo sqlite_omit_load_extension' -ldflags '-linkmode external -extldflags -static' -o "$output/bin/picketx-controller" ./cmd/picketx-controller
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -o "$output/bin/picketx" ./cmd/picketx
for binary in "$output/bin/picketx-controller" "$output/bin/picketx"; do
  file "$binary"
  headers=$(readelf -l "$binary"); dynamic=$(readelf -d "$binary")
  if [[ "$headers" == *INTERP* || "$dynamic" == *NEEDED* ]]; then
    echo 'Refusing dynamically linked bundle artifact' >&2
    exit 1
  fi
done
npm ci --prefix apps/web
npm run typecheck --prefix apps/web
npm run lint --prefix apps/web
npm run build --prefix apps/web
if [[ -n $(find apps/web/dist -type l -print -quit) ]]; then
  echo 'Refusing symlink in built Web assets' >&2
  exit 1
fi
cp -R apps/web/dist "$output/web"
cp LICENSE "$output/LICENSE"
cp deploy/controller/README.md deploy/controller/README.zh-CN.md deploy/controller/picketx-controller.service "$output/deploy/"
printf '{"commit":"%s","tree":"%s","platform":"linux-amd64"}\n' "$(git rev-parse HEAD)" "$(git rev-parse 'HEAD^{tree}')" > "$output/SOURCE.json"
(
  cd "$output"
  find bin web deploy -type f -print0 | sort -z | xargs -0 sha256sum > SHA256SUMS
  sha256sum LICENSE SOURCE.json >> SHA256SUMS
  sha256sum --check SHA256SUMS
)
echo "Built standalone Controller bundle: $output"
