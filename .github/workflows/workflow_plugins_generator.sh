#!/bin/bash

plugins=("beego"  "chi"  "dotweb"  "echo"  "fiber"  "gin"  "go-zero"  "kratos"  "roadrunner"  "skipper"  "souin"  "traefik"  "tyk"  "webgo")

IFS= read -r -d '' tpl <<EOF
name: Build and validate Souin as plugins

on:
  - pull_request

jobs:
  build-caddy-validator:
    name: Check that Souin build as caddy module
    runs-on: ubuntu-latest
    steps:
      -
        name: Add domain.com host to /etc/hosts
        run: |
          sudo echo "127.0.0.1 domain.com" | sudo tee -a /etc/hosts
      -
        name: Install Go
        uses: actions/setup-go@v2
        with:
          go-version: 1.19
      -
        name: Checkout code
        uses: actions/checkout@v2
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
EOF
  workflow+="$tpl"
done
echo "${workflow%$'\n'}" >  "$( dirname -- "$0"; )/plugins.yml"