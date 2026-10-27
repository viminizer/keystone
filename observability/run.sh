#!/bin/bash
# Starts keystone, Prometheus, Grafana and the k6 load in the background,
# and opens Grafana. Everything keeps running until it is stopped with:
#
#   ./observability/run.sh          start everything
#   ./observability/run.sh --down   stop everything

set -e
cd "$(dirname "$0")/.."

if [ "$1" = "--down" ]; then
  pkill -f "k6 run observability/load.js" && echo "stopped the load" || true
  keystone=$(lsof -t -iTCP:6380 -sTCP:LISTEN || true)
  if [ -n "$keystone" ]; then
    kill $keystone
    echo "stopped keystone"
  fi
  docker compose -f observability/docker-compose.yml down
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
KEYSTONE_AOF_PATH="$data/appendonly.aof" nohup "$data/keystone" > /dev/null 2> "$data/keystone.log" &
keystone=$!

until nc -z localhost 6380 2>/dev/null; do
  kill -0 $keystone
  sleep 0.5
done
until curl -sf -o /dev/null localhost:3000/api/health; do
  sleep 1
done

nohup k6 run observability/load.js > /dev/null 2> "$data/k6.log" &

open http://localhost:3000
echo "keystone logs: $data/keystone.log"
echo "k6 logs: $data/k6.log"
echo "stop everything with: ./observability/run.sh --down"
