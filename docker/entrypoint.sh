#!/bin/sh
set -e

PUBLIC_PORT="${PORT:-10000}"

cd /app
npx prisma migrate deploy

PORT=8000 relay &
RELAY_PID=$!
PORT=3000 node dist/index.js &
API_PID=$!
PORT="$PUBLIC_PORT" caddy run --config /etc/caddy/Caddyfile --adapter caddyfile &
CADDY_PID=$!

stopping=0
shutdown() {
	if [ "$stopping" -eq 1 ]; then
		return
	fi
	stopping=1
	kill -TERM "$CADDY_PID" "$API_PID" "$RELAY_PID" 2>/dev/null || true
}
trap shutdown INT TERM

while kill -0 "$RELAY_PID" 2>/dev/null && kill -0 "$API_PID" 2>/dev/null && kill -0 "$CADDY_PID" 2>/dev/null; do
	sleep 5
done

shutdown
wait 2>/dev/null || true
