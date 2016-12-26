#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

LIB_ROOT=$(dirname "${BASH_SOURCE}")/..
source "$LIB_ROOT/libbuild/common/lib.sh"
source "$LIB_ROOT/libbuild/common/public_image.sh"

GOPATH=$(go env GOPATH)
SRC=$GOPATH/src
BIN=$GOPATH/bin
ROOT=$GOPATH
REPO_ROOT=$GOPATH/src/github.com/appscode/g2

APPSCODE_ENV=${APPSCODE_ENV:-dev}

IMG=gearmand
# TAG=0.1.0
if [ -f "$REPO_ROOT/dist/.tag" ]; then
	export $(cat $REPO_ROOT/dist/.tag | xargs)
fi

build_binary() {
	pushd $REPO_ROOT
	./hack/builddeps.sh
    ./hack/make.py build gearmand
	detect_tag $REPO_ROOT/dist/.tag
	popd
}

build_docker() {
	pushd $REPO_ROOT/hack/docker
	cp $REPO_ROOT/dist/gearmand/gearmand-linux-amd64 gearmand
	chmod 755 gearmand

	gsutil cp gs://appscode-dev/binaries/gcron/0.3.0/gcron-linux-amd64 gcron
	chmod 755 gcron

	local cmd="docker build -t appscode/$IMG:$TAG ."
	echo $cmd; $cmd

	rm gearmand gcron
	popd
}

build() {
	build_binary
	build_docker
}

source_repo $@
