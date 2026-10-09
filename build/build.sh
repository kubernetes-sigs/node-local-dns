#!/bin/bash
# Copyright 2016 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -o errexit
set -o nounset
set -o pipefail

if [ -z "${PKG}" ]; then
    echo "PKG must be set"
    exit 1
fi
if [ -z "${ARCH}" ]; then
    echo "ARCH must be set"
    exit 1
fi
if [ -z "${VERSION}" ]; then
    echo "VERSION must be set"
    exit 1
fi

export CGO_ENABLED=0
export GOOS="linux"
export GOARCH="${ARCH}"

# On a native build (host arch == target arch) Go outputs to $GOPATH/bin/ by
# default, so we redirect explicitly to match the volume-mount path used by
# rules.mk. On a cross-compilation (e.g. arm64 host building for amd64) Go
# already places binaries in $GOPATH/bin/linux_${GOARCH}/, so GOBIN must not
# be set (Go rejects GOBIN during cross-compilation).
GOHOSTARCH=$(go env GOHOSTARCH)
if [ "${GOARCH}" == "${GOHOSTARCH}" ]; then
    export GOBIN="$GOPATH/bin/linux_${GOARCH}"
fi

go install                                                         \
    -installsuffix "static"                                        \
    -ldflags "-X ${PKG}/pkg/version.VERSION=${VERSION}"            \
    ./...
