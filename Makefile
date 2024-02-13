OUTPUT := bin/${current_dir}

build:
	CGO_ENABLED=${CGO_ENABLED} go build ${LDFLAGS} -o ${OUTPUT} *.go

.PHONY: lint
lint:
	go vet -tags test $$(go list ./...)
	golangci-lint run

test:
	go test -race -cover -coverprofile cover.out $$(go list ./...)

tidy:
	go mod tidy

fmt:
	go fmt ./...
	find . -name '*.go' -exec gci write -s 'standard' -s 'default' -s 'prefix(github.com/gametimesf/ecs-deploy)' {} \; > /dev/null

