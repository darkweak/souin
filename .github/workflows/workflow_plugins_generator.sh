#!/bin/bash

plugins=("beego"  "chi"  "dotweb"  "echo"  "fiber"  "gin"  "go-zero"  "goyave"  "kratos"  "roadrunner"  "skipper"  "souin"  "traefik"  "tyk"  "webgo")
durations=("35"   "35"   "35"      "35"    "45"     "40"   "50"       "40"      "50"      "10"          "65"       "40"     "20"       "30"   "45")
versions=("18"    "18"   "18"      "18"    "18"     "18"   "18"       "18"      "18"      "18"          "18"       "18"     "18"       "16"   "18")

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
          go-version: 1.18
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
    name: Check that Souin build as $capitalized middleware
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
          go-version: 1.${versions[$i]}
      -
        name: Checkout code
        uses: actions/checkout@v2
      -
        name: Run $capitalized tests
        run: cd plugins/$lower && go test -v .
      -
        name: Build Souin as $capitalized plugin
        run: make build-and-run-$lower
        env:
          GH_APP_TOKEN: \${{ secrets.GH_APP_TOKEN }}
      -
        name: Wait for Souin is really loaded inside $capitalized as middleware
        uses: jakejarvis/wait-action@master
        with:
          time: ${durations[$i]}s
      -
        name: Set $capitalized logs configuration result as environment variable
        run: cd plugins/$lower && echo "\$(make load-checker)" >> \$GITHUB_ENV
      -
        name: Check if the configuration is loaded to define if Souin is loaded too
        uses: nick-invision/assert-action@v1
        with:
          expected: '"Souin configuration is now loaded."'
          actual: \${{ env.MIDDLEWARE_RESULT }}
          comparison: contains
      -
        name: Run $capitalized E2E tests
        uses: anthonyvscode/newman-action@v1
        with:
          collection: "docs/e2e/Souin E2E.postman_collection.json"
          folder: $capitalized
          reporters: cli
          delayRequest: 5000
EOF
  workflow+="$tpl"
done
echo "${workflow%$'\n'}" >  "$( dirname -- "$0"; )/plugins.yml"