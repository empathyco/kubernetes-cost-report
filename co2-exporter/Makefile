BINARY_NAME=cost_report

build_all: build_darwin build_linux build_windows

build_darwin:
	GOARCH=amd64 GOOS=darwin go build -o ${BINARY_NAME}-darwin main.go

build_linux:
	GOARCH=amd64 GOOS=linux go build -o ${BINARY_NAME}-linux main.go

build_windows:
	GOARCH=amd64 GOOS=window go build -o ${BINARY_NAME}-windows main.go

lint:
	golangci-lint run  --tests=false --exclude-use-default=false --enable-all  -D gci -D scopelint -D exhaustivestruct  -D wsl -D gomnd -D gochecknoglobals -D lll -D golint -D ireturn -D tagliatelle -D contextcheck -D forcetypeassert

test:
	go test -v ./...

test-ci:
	CI=test go test -v ./...

test-coverage:
	go test -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out
	

build-and-run: build run


docker-build:
	docker build -t ${BINARY_NAME} .

docker-run:
	docker run -it --rm -p 8080:8080 ${BINARY_NAME}

docker-build-and-run: docker-build docker-run

clean:
	go clean
	rm ${BINARY_NAME}-darwin
	rm ${BINARY_NAME}-linux
	rm ${BINARY_NAME}-windows