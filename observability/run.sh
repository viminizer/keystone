#!/bin/bash
# Starts keystone, Prometheus, Grafana and the k6 load in the background,
# and opens Grafana. Everything keeps running until it is stopped.
#
#   ./observability/run.sh            start everything, with no saved keys
#   ./observability/run.sh --restart  build and restart only keystone
#   ./observability/run.sh --down     stop everything

set -e
cd "$(dirname "$0")/.."

# The append-only file and the logs. --restart keeps the file, so keystone
# loads the same keys again.
data=observability/data

build_keystone() {
  mkdir -p "$data"
  go build -o "$data/keystone.new" ./cmd/keystone
}

start_keystone() {
  # The new program is moved over the old one, because changing a running
  # program's file in place can crash it.
  mv "$data/keystone.new" "$data/keystone"
  KEYSTONE_AOF_PATH="$data/appendonly.aof" nohup "$data/keystone" > /dev/null 2>> "$data/keystone.log" &
  keystone=$!
  until nc -z localhost 6380 2>/dev/null; do
    kill -0 $keystone
    sleep 0.2
  done
  echo "started keystone"
}

stop_keystone() {
  keystone=$(lsof -t -iTCP:6380 -sTCP:LISTEN || true)
  if [ -n "$keystone" ]; then
    kill $keystone
    # Wait until it has saved the file and let go of the port.
    while kill -0 $keystone 2>/dev/null; do
      sleep 0.2
    done
    echo "stopped keystone"
  fi
}

if [ "$1" = "--down" ]; then
  pkill -f "k6 run observability/load.js" && echo "stopped the load" || true
  stop_keystone
  docker compose -f observability/docker-compose.yml down
  exit 0
fi

if [ "$1" = "--restart" ]; then
  # Build first, so a build error does not leave keystone stopped.
  build_keystone
  stop_keystone
  start_keystone
  exit 0
fi

if nc -z localhost 6380 2>/dev/null; then
  echo "port 6380 is in use, stop the other keystone first"
  exit 1
fi

docker compose -f observability/docker-compose.yml up -d

rm -rf "$data"
mkdir -p "$data"
build_keystone
start_keystone

until curl -sf -o /dev/null localhost:3000/api/health; do
  sleep 1
done

nohup k6 run observability/load.js > /dev/null 2> "$data/k6.log" &

open http://localhost:3000
echo "logs: $data/keystone.log and $data/k6.log"
echo "stop everything with: ./observability/run.sh --down"
