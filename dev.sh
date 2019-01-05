#! /bin/sh

trap 'echo "Killing background jobs..."; kill $(jobs -p)' EXIT

(cd webapp && make update-version && npm start) &

while true; do
    find go -name \*.go | grep -v packr | \
	entr -d -r sh -c "(cd go && make) && ./go/maker server"
done

kill $(jobs -p)

