VERSION=1.0.0
LINKER_FLAGS=-X github.com/AlexanderRichey/yagss/internal/version.Version=${VERSION}
# RELEASE_BUILD_LINKER_FLAGS disables DWARF and symbol table generation to reduce binary size
RELEASE_BUILD_LINKER_FLAGS=-s -w

local: gen
	mkdir -p build/bin
	go build -ldflags "${LINKER_FLAGS}" -o build/bin/yagss cmd/yagss/main.go

install:
	@cd cmd/yagss && go install -ldflags "${LINKER_FLAGS}"

compile-all: compile-linux-amd64 compile-linux-arm compile-darwin-amd64 compile-darwin-arm

compile-linux-amd64:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "${LINKER_FLAGS} ${RELEASE_BUILD_LINKER_FLAGS}" -o build/bin/yagss-linux-amd64 ./cmd/yagss

compile-linux-arm:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm go build -ldflags "${LINKER_FLAGS} ${RELEASE_BUILD_LINKER_FLAGS}" -o build/bin/yagss-linux-arm ./cmd/yagss

compile-darwin-amd64:
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags "${LINKER_FLAGS} ${RELEASE_BUILD_LINKER_FLAGS}" -o build/bin/yagss-darwin-amd64 ./cmd/yagss

compile-darwin-arm:
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags "${LINKER_FLAGS} ${RELEASE_BUILD_LINKER_FLAGS}" -o build/bin/yagss-darwin-arm ./cmd/yagss

.PHONY: gen
gen:
	go generate ./...

.PHONY: clean
clean:
	rm -rf build/*
	rm -rf internal/proj/data

.PHONY: test
test:
	go test -v -cover ./...
