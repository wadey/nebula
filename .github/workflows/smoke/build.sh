#!/bin/sh

set -e -x

rm -rf ./build
mkdir ./build

(
    cd build

    cp ../../../../build/linux-amd64/nebula .
    cp ../../../../build/linux-amd64/nebula-cert .

    HOST="lighthouse1" \
        AM_LIGHTHOUSE=true \
        ../genconfig.sh >lighthouse1.yml

    HOST="host2" \
        LIGHTHOUSES="192.168.100.1 172.17.0.2:4242" \
        ../genconfig.sh >host2.yml

    HOST="host3" \
        LIGHTHOUSES="192.168.100.1 172.17.0.2:4242" \
        INBOUND='[{"port": "any", "proto": "icmp", "group": "lighthouse"}]' \
        ../genconfig.sh >host3.yml

    HOST="host4" \
        LIGHTHOUSES="192.168.100.1 172.17.0.2:4242" \
        OUTBOUND='[{"port": "any", "proto": "icmp", "group": "lighthouse"}]' \
        ../genconfig.sh >host4.yml

    ../../../../nebula-cert ca -name "Smoke Test"
    ../../../../nebula-cert sign -name "lighthouse1" -groups "lighthouse,lighthouse1" -ip "192.168.100.1/24"
    ../../../../nebula-cert sign -name "host2" -groups "host,host2" -ip "192.168.100.2/24"
    ../../../../nebula-cert sign -name "host3" -groups "host,host3" -ip "192.168.100.3/24"
    ../../../../nebula-cert sign -name "host4" -groups "host,host4" -ip "192.168.100.4/24"
)

sudo docker build -t nebula:smoke .
