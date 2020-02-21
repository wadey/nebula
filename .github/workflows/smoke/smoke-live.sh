#!/bin/bash

set -e

redact() {
    sed "s/$SMOKE_HOST:$SMOKE_PORT/SMOKE_HOST:SMOKE_PORT/g"
}

rm -rf ./build
mkdir ./build

cd build

cp ../../../../nebula .
cp ../../../../nebula-cert .

RAND1="$((RANDOM % 254 + 1))"
RAND2="$((RANDOM % 254 + 1))"
VPNIP="192.168.$RAND1.$RAND2"

HOST="host" LIGHTHOUSES="192.168.0.1 $SMOKE_HOST:$SMOKE_PORT" ../genconfig.sh >host.yml

echo "$SMOKE_CA_CRT" >ca.crt
echo "$SMOKE_PROV_CRT" >prov.crt
echo "$SMOKE_PROV_KEY" >prov.key

./nebula-cert sign -ca-crt prov.crt -ca-key prov.key -name "host" -ip "$VPNIP/16"

nebula_timeout() {
    ./nebula -config host.yml 2>&1 &
    NPID="$!"
    sleep 10
    kill "$NPID"
}

(nebula_timeout | redact) &

sleep 1

ping -c 1 -W 10 192.168.0.1

wait
