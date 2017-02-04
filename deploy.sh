#!/bin/bash
echo "embedding assets" &&
go generate ./castle/... &&
echo "building" &&
GOOS=linux GOARCH=arm go build -ldflags "-w -s -X main.COMMIT=$(git rev-parse HEAD | cut -c 1-8) -X main.BUILDTIME=$(date -u +%s)" -o /tmp/gobin &&
echo "uploading" &&
rsync -e "ssh -p 22" --compress /tmp/gobin root@jamopi:/usr/local/bin/castlebot &&
echo "done" &&
rm /tmp/gobin
