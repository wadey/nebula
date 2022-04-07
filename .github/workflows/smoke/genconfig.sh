#!/bin/sh

set -e

FIREWALL_ALL='[{"port": "any", "proto": "any", "host": "any"}]'

if [ "$STATIC_HOSTS" ] || [ "$LIGHTHOUSES" ]
then
  echo "static_host_map:"
  echo "$STATIC_HOSTS" | while read -r NEBULA_IP STATIC
  do
    [ -z "$NEBULA_IP" ] || echo "  '$NEBULA_IP': ['$STATIC']"
  done
  echo "$LIGHTHOUSES" | while read -r NEBULA_IP STATIC
  do
    [ -z "$NEBULA_IP" ] || echo "  '$NEBULA_IP': ['$STATIC']"
  done
  echo
fi

lighthouse_hosts() {
  if [ "$LIGHTHOUSES" ]
  then
    echo
    echo "$LIGHTHOUSES" | while read -r NEBULA_IP STATIC
    do
      echo "    - '$NEBULA_IP'"
    done
  else
    echo "[]"
  fi
}

cat <<EOF
pki:
  ca: ca.crt
  cert: ${HOST}.crt
  key: ${HOST}.key

lighthouse:
  am_lighthouse: ${AM_LIGHTHOUSE:-false}
  hosts: $(lighthouse_hosts)

listen:
  host: 0.0.0.0
  port: ${LISTEN_PORT:-4242}

tun:
  dev: ${TUN_DEV:-nebula1}

firewall:
  outbound: ${OUTBOUND:-$FIREWALL_ALL}
  inbound: ${INBOUND:-$FIREWALL_ALL}
EOF
