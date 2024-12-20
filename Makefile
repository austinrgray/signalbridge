SHELL := /bin/bash

build:
	go build -o bin/app

run: build
	./bin/app

test:
	go test -test.timeout=10m0s -count=1