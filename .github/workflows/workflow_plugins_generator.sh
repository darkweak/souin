#!/bin/bash

plugins=("beego"  "chi"  "dotweb"  "echo"  "fiber"  "gin"  "goa"  "go-zero"  "hertz"  "kratos"  "roadrunner"  "skipper"  "souin"  "traefik"  "tyk"  "webgo")
go_version=1.20

IFS= read -r -d '' tpl <<EOF
name: Build and validate Souin as plugins

on:
  - pull_request

jobs:
  build-caddy-validator:
    name: Check that Souin build as caddy module
    runs-on: ubuntu-latest
    services:
      redis:
        image: redis
        ports:
          - 6379:6379
      etcd:
        image: quay.io/coreos/etcd
        env:
          ETCD_NAME: etcd0
          ETCD_ADVERTISE_CLIENT_URLS: http://etcd:2379,http://etcd:4001
          ETCD_LISTEN_CLIENT_URLS: http://0.0.0.0:2379,http://0.0.0.0:4001
          ETCD_INITIAL_ADVERTISE_PEER_URLS: http://etcd:2380
          ETCD_LISTEN_PEER_URLS: http://0.0.0.0:2380
          ETCD_INITIAL_CLUSTER_TOKEN: etcd-cluster-1
          ETCD_INITIAL_CLUSTER: etcd0=http://etcd:2380
          ETCD_INITIAL_CLUSTER_STATE: new
        ports:
          - 2379:2379
          - 2380:2380
          - 4001:4001
    steps:
      -
        name: Add domain.com host to /etc/hosts
        run: |
          sudo echo "127.0.0.1 domain.com etcd redis" | sudo tee -a /etc/hosts
      -
        name: Install Go
        uses: actions/setup-go@v3
        with:
          go-version: '$go_version'
      -
        name: Checkout code
        uses: actions/checkout@v3
      -
        name: Install xcaddy
        run: go install github.com/caddyserver/xcaddy/cmd/xcaddy@latest
      -
        name: Build Souin as caddy module
        run: cd plugins/caddy && xcaddy build --with github.com/darkweak/souin/plugins/caddy=./ --with github.com/darkweak/souin@latest=../..
      -
        name: Run Caddy tests
        run: cd plugins/caddy && go test -v ./...
      -
        name: Run detached caddy
        run: cd plugins/caddy && ./caddy run &
      -
        name: Run Caddy E2E tests
        uses: anthonyvscode/newman-action@v1
        with:
          collection: "docs/e2e/Souin E2E.postman_collection.json"
          folder: Caddy
          reporters: cli
          delayRequest: 5000
      -
        name: Run detached caddy
        run: cd plugins/caddy && ./caddy stop
      -
        name: Run detached caddy
        run: cd plugins/caddy && ./caddy run --config ./configuration.json &
      -
        name: Run Caddy E2E tests
        uses: anthonyvscode/newman-action@v1
        with:
          collection: "docs/e2e/Souin E2E.postman_collection.json"
          folder: Caddy
          reporters: cli
          delayRequest: 5000
EOF
workflow+="$tpl"

for i in ${!plugins[@]}; do
  lower="${plugins[$i]}"
  capitalized="$(tr '[:lower:]' '[:upper:]' <<< ${lower:0:1})${lower:1}"
  IFS= read -d '' tpl <<EOF
  build-$lower-validator:
    name: Check that Souin build as middleware
    uses: ./.github/workflows/plugin_template.yml
    secrets: inherit
    with:
      CAPITALIZED_NAME: $capitalized
      LOWER_NAME: $lower
      GO_VERSION: '$go_version'
EOF
  workflow+="$tpl"
done
echo "${workflow%$'\n'}" >  "$( dirname -- "$0"; )/plugins.yml"