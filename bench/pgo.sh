#!/usr/bin/env bash
# Profiles the benchmarks and writes cmd/targascript/default.pgo, which Go
# uses to optimize the targascript binary for them (profile-guided
# optimization). Run it again after big changes to the VM.
set -e
cd "$(dirname "$0")/.."
dir=$(mktemp -d)
trap 'rm -rf "$dir"' EXIT
go build -o "$dir/targascript" ./cmd/targascript
# Each run is short, so each benchmark runs several times for enough samples
for script in bench/*.tg; do
  for i in 1 2 3 4 5 6 7 8 9 10; do
    TARGASCRIPT_CPUPROFILE="$dir/$(basename "$script" .tg)-$i.prof" "$dir/targascript" "$script" > /dev/null
  done
done
go tool pprof -proto "$dir"/*.prof > cmd/targascript/default.pgo
echo "wrote cmd/targascript/default.pgo"
