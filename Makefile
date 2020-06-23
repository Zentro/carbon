VERSION ?= $(shell git describe --tags --always --dirty --match=v* 2> /dev/null || echo "0.0.1")
BUILD_TIME = `date +%FT%T%z`
LDFLAGS := -ldflags "-X system.Version=${VERSION} -X system.BuildTime=${BUILD_TIME}"
TAGS := -tags=jsoniter

.PHONY: build
build:
	GOOS=linux GOARCH=amd64 go build ${LDFLAGS} -o build/carbon_linux_amd64 -v carbon.go
	GOOS=linux GOARCH=arm64 go build ${LDFLAGS} -o build/carbon_linux_arm64 -v carbon.go

.PHONY: compress
compress:
	upx --brute build/carbon_*

.PHONY: clean
clean:
	rm -rf build/carbon_*