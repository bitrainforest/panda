.PHONY: clean swagger build

all: clean swagger build

swagger:
	swag init -g api.go -d ./pkg/api

env:
	source ${PWD}/scripts/env.sh && pwd

build:
	${PWD}/scripts/build.sh

clean:
	rm -rf ${PWD}/output
