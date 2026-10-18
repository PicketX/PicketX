#!/bin/sh
# SPDX-License-Identifier: Apache-2.0
set -eu
[ "$(protoc --version)" = 'libprotoc 31.1' ]
[ "$(protoc-gen-go --version)" = 'protoc-gen-go v1.36.11' ]
[ "$(protoc-gen-go-grpc --version)" = 'protoc-gen-go-grpc 1.6.1' ]
protoc --go_out=. --go_opt=module=github.com/PicketX/PicketX \
  --go-grpc_out=. --go-grpc_opt=module=github.com/PicketX/PicketX \
  proto/picketx/agent/v1alpha1/sync.proto
