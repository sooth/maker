#! /bin/sh

IMAGE="crankykernel/maker:builder"

docker_build() {
    docker build ${CACHE_FROM} -t "${IMAGE}" -f build/Dockerfile .
}

docker_run() {
    # Do we have a tty?
    it=""
    if [ -t 1 ] ; then
	it="-it"
    fi

    cache="$(pwd)/.docker_cache"

    mkdir -p ${cache}/go
    mkdir -p ${cache}/node_modules
    mkdir -p ${cache}/npm
    mkdir -p ./webapp/node_modules

    real_uid=$(id -u)
    real_gid=$(id -g)

    if [[ "${real_uid}" = "0" ]]; then
	image_home="/root"
    else
	image_home="/home/builder"
    fi

    volumes=""
    volumes="-v $(pwd):/src"
    
    volumes="${volumes} -v ${cache}/go:${image_home}/go"
    volumes="${volumes} -v ${cache}/npm:${image_home}/npm"
    volumes="${volumes} -v ${cache}/node_modules:/src/webapp/node_modules"

    docker run \
	   --rm ${it} \
	   ${volumes} \
	   -e REAL_UID=$(id -u) \
	   -e REAL_GID=$(id -g) \
	   -w /src \
	   ${IMAGE} "$1"
}

docker_build

docker_run "make install-deps"
docker_run "GOOS=linux GOARCH=amd64 make dist"
docker_run "CC=x86_64-w64-mingw32-gcc GOOS=windows GOARCH=amd64 make dist"
docker_run "CC=o64-clang GOOS=darwin GOARCH=amd64 make dist"
