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
VPNIP="198.18.$RAND1.$RAND2"

HOST="host" \
    LIGHTHOUSES="198.18.0.1 $SMOKE_HOST:$SMOKE_PORT" \
    UNSAFE_ROUTE="192.0.2.0/24" \
    UNSAFE_VIA="198.18.0.1" \
    ../genconfig.sh >host.yml

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

sleep 5

if [ "$(uname)" = "Linux" ]
then
    ping -c 1 -w 5 198.18.0.1
    ping -c 1 -w 5 192.0.2.1
elif [ "$(uname)" = "Darwin" ]
then
    ping -c 1 -t 5 198.18.0.1
    ping -c 1 -t 5 192.0.2.1
else
    # Windows
    ping -n 1 -w 5000 198.18.0.1
    ping -n 1 -w 5000 192.0.2.1
fi

echo
echo

wait
