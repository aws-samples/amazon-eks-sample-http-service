#!/bin/sh

set -ex

mkdir -p /var/cache/nginx

get_metadata() {
    metadata_url="http://169.254.169.254/latest/meta-data/$1"; shift
    file="$1"; shift

    if ! curl -sf -o "$file" "$metadata_url"; then
        if [ -z "$TOKEN" ]; then
            TOKEN=`curl -sf -X PUT "http://169.254.169.254/latest/api/token" -H "X-aws-ec2-metadata-token-ttl-seconds: 21600"`
        fi
        curl -H "X-aws-ec2-metadata-token: $TOKEN" -sSf -o "$file" "$metadata_url"
    fi
}

# /run must be mounted read-write via tmpfs
get_metadata instance-id /run/instance-id.txt
get_metadata placement/availability-zone /run/availability-zone.txt

