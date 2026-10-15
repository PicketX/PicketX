#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
set -euo pipefail
cd "$(dirname "$0")/.."
task_tmp="$(mktemp -d)"
trap 'rm -rf "$task_tmp"' EXIT
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -o "$task_tmp/picketx" ./cmd/picketx
file "$task_tmp/picketx"
headers="$(readelf -l "$task_tmp/picketx")"
dynamic="$(readelf -d "$task_tmp/picketx")"
if [[ "$headers" == *INTERP* || "$dynamic" == *NEEDED* ]]; then
  echo 'CLI has an unexpected dynamic loader or shared-library dependency' >&2
  exit 1
fi
for hour in 09 10 12; do
  "$task_tmp/picketx" config check --file examples/static/gitea.yaml --facts examples/static/facts.json --at "2026-10-15T${hour}:00:00Z" > "$task_tmp/at${hour}.json"
done
"$task_tmp/picketx" config check --file examples/static/empty.yaml --facts examples/static/facts.json --at 2026-10-15T09:00:00Z > "$task_tmp/empty.json"
python3 - "$task_tmp" <<'PY'
import json, pathlib, sys
root = pathlib.Path(sys.argv[1])
for hour, count in [('09', 2), ('10', 1), ('12', 0)]:
    result = json.loads((root / ('at' + hour + '.json')).read_text())
    assert result['state'] == 'validated' and result['enforcementConfirmed'] is False
    assert len(result['projections']) == 1
    projection = result['projections'][0]
    assert len(projection['Allow'] or []) == count
    assert len(projection['Protected']) == 2
    print(hour + ':00Z:', count, 'allow, 2 protected')
result = json.loads((root / 'empty.json').read_text())
assert len(result['projections']) == 1 and not result['projections'][0]['Protected']
assert result['enforcementConfirmed'] is False
print('Explicit withdrawal: known agent, 0 protected; no enforcement claim')
PY
