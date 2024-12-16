SHELL := /bin/bash

build:
	go build -o bin/app

run: build
	./bin/app

test:
	go test -count=1