#!/bin/bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a \
  -tags netgo -ldflags '-w -extldflags "-static"' -o /app/cmd/main .
chmod +x /app/cmd/main
cp /app/cmd/main /usr/local/bin/
/usr/local/bin/main
