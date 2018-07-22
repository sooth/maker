#! /bin/sh

#
# Requires reflex:
#    go get github.com/cespare/reflex
#

set -x
set -e

trap 'echo "Killing background jobs..."; kill $(jobs -p)' EXIT

args="$@"
command=""

case "$1" in
    -*)
	command=server
	;;
esac

(cd webapp && make serve) &
reflex -s -R -packr\.go -r \.go$ -- make dev-server
