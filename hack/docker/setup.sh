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

clean() {
	pushd $REPO_ROOT/hack/docker
	rm gearmand Dockerfile
	popd
}

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

	cat >Dockerfile <<EOL
FROM alpine

RUN set -x \
  && apk update \
  && apk add ca-certificates \
  && rm -rf /var/cache/apk/*

COPY gearmand /gearmand

USER nobody:nobody
ENTRYPOINT ["/gearmand"]
EOL
	local cmd="docker build -t appscode/$IMG:$TAG ."
	echo $cmd; $cmd

	rm gearmand Dockerfile
	popd
}

build() {
	build_binary
	build_docker
}

docker_push() {
	if [ "$APPSCODE_ENV" = "prod" ]; then
		echo "Nothing to do in prod env. Are you trying to 'release' binaries to prod?"
		exit 0
	fi

    if [[ "$(docker images -q appscode/$IMG:$TAG 2> /dev/null)" != "" ]]; then
        docker push appscode/$IMG:$TAG
    fi
}

docker_release() {
	if [ "$APPSCODE_ENV" != "prod" ]; then
		echo "'release' only works in PROD env."
		exit 1
	fi
	if [ "$TAG_STRATEGY" != "git_tag" ]; then
		echo "'apply_tag' to release binaries and/or docker images."
		exit 1
	fi

    if [[ "$(docker images -q appscode/$IMG:$TAG 2> /dev/null)" != "" ]]; then
        docker push appscode/$IMG:$TAG
    fi
}

source_repo $@
