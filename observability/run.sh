#!/bin/bash
# Runs keystone, Prometheus, Grafana and the k6 load with one command:
#
#   ./observability/run.sh
#
# Keystone stops when the load ends, or on Ctrl+C. Prometheus and Grafana
# keep running, so the graphs can still be looked at. Stop everything with:
#
#   ./observability/run.sh --down

set -e
cd "$(dirname "$0")/.."

if [ "$1" = "--down" ]; then
  docker compose -f observability/docker-compose.yml down
  keystone=$(lsof -t -iTCP:6380 -sTCP:LISTEN || true)
  if [ -n "$keystone" ]; then
    kill $keystone
    echo "stopped keystone"
  fi
  exit 0
fi

if nc -z localhost 6380 2>/dev/null; then
  echo "port 6380 is in use, stop the other keystone first"
  exit 1
fi

docker compose -f observability/docker-compose.yml up -d

# Each run starts with an empty append-only file.
data=$(mktemp -d)
go build -o "$data/keystone" ./cmd/keystone
KEYSTONE_AOF_PATH="$data/appendonly.aof" "$data/keystone" 2> "$data/keystone.log" &
keystone=$!
trap 'kill $keystone; wait $keystone' EXIT
echo "keystone logs: $data/keystone.log"

until nc -z localhost 6380 2>/dev/null; do
  kill -0 $keystone
  sleep 0.5
done
until curl -sf -o /dev/null localhost:3000/api/health; do
  sleep 1
done

open http://localhost:3000
k6 run observability/load.js
