OUTPUT = bin/ecs-deploy
BUILD_DATE=`date +%FT%T%z`
GIT_HASH=`git rev-parse HEAD`
LDFLAGS=-ldflags="-X github.com/gametimesf/log.builtAt=${BUILD_DATE} -X github.com/gametimesf/log.commitHash=${GIT_HASH}"

build:
	CGO_ENABLED=0 go build ${LDFLAGS} -o ${OUTPUT} *.go

.PHONY: lint
lint:
	golangci-lint run

test:
	go test -coverprofile tests.cover.tmp -race -cover ./...
	grep -v mock < tests.cover.tmp > tests.cover
	rm tests.cover.tmp

tidy:
	go mod tidy

fmt:
	go fmt ./...
	find . -name '*.go' -exec gci write -s 'standard' -s 'default' -s 'prefix(github.com/gametimesf/ecs-deploy)' {} \; > /dev/null
