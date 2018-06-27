#! /bin/sh

DOCKER_TAG="maker/build"

docker build -t ${DOCKER_TAG} -f build/Dockerfile .
docker run -v $(pwd)/dist:/dist --rm -it -t ${DOCKER_TAG}
