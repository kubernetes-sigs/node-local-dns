#!/bin/bash
# This script runs some very basic commands to ensure that the newly build
# images are working correctly. Invoke as:
# ./image-checks.sh <image-tag> <registry-name>
TAG=$1
REGISTRY=${2:-registry.k8s.io/dns}
echo "Verifying that iptables exists in node-cache image"
docker run --rm -it --entrypoint=iptables ${REGISTRY}/k8s-dns-node-cache:${TAG}
echo "Verifying that node-cache binary exists in node-cache image"
docker run --rm -it --entrypoint=/node-cache ${REGISTRY}/k8s-dns-node-cache:${TAG}
