COMMIT_SHA=$(shell git log --pretty=format:"%h" -1 || echo "unknown")
TAG=$(shell git describe --tags || echo "unknown")
BUILD_TIME=$(shell date '+%Y-%m-%d %H:%M:%S')

all: build

build: rcproxy

rcproxy:
	go build -ldflags "-X \"main.CommitSHA=${COMMIT_SHA}\" -X \"main.Tag=${TAG}\" -X \"main.BuildTime=${BUILD_TIME}\"" -o ./bin/rcproxy main.go

clean:
	@rm -rf bin

test:
	go test ./core/...