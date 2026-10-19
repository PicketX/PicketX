#!/bin/sh
# SPDX-License-Identifier: Apache-2.0
set -eu
# rc6 reports only v2.0.0 via --version; inspect embedded module metadata instead.
go version -m "$(command -v swag)" | awk '$1 == "mod" && $2 == "github.com/swaggo/swag/v2" && $3 == "v2.0.0-rc6" { found=1 } END { exit !found }'
swag init --v3.1 --generalInfo openapi.go --dir internal/controller,internal/publication,internal/management,internal/agentsync,internal/accessrequest,pkg/configuration --parseInternal --outputTypes json --output docs/api
node scripts/normalize-management-api.mjs
