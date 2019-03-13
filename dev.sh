#! /bin/sh

trap 'echo "Killing background jobs..."; kill $(jobs -p)' EXIT

make

(cd webapp && make update-version && npm start) &

args="$@"

while true; do
    find go -name \*.go | grep -v packr | \
	entr -d -r sh -c "(cd go && make) && ./go/maker server ${args}"
done

kill $(jobs -p)

