BINARY ?= oc-nodepp
GOOS?=linux
GOARCH?=amd64
GOENV=GOOS=${GOOS} GOARCH=${GOARCH} CGO_ENABLED=0 GOFLAGS=
GO_SOURCES := $(find $(CURDIR) -type f -name "*.go" -print)

default: build

clean:
	rm -f ${BINARY}

build: clean $(BINARY)

$(BINARY): $(GO_SOURCES)
	${GOENV} go build -o ${BINARY} main.go

run:
	go run main.go

