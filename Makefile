VERSION=1.0.0
LINKER_FLAGS=-X github.com/AlexanderRichey/yagss/internal/version.Version=${VERSION}
# RELEASE_BUILD_LINKER_FLAGS disables DWARF and symbol table generation to reduce binary size
RELEASE_BUILD_LINKER_FLAGS=-s -w

local: gen
	mkdir -p build/bin
	go build -ldflags "${LINKER_FLAGS}" -o build/bin/yagss cmd/yagss/main.go

install: gen
	go install -ldflags "${LINKER_FLAGS}" cmd/yagss/main.go

compile-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "${LINKER_FLAGS} ${RELEASE_BUILD_LINKER_FLAGS}" -o build/bin/yagss-linux-amd64 ./cmd/yagss

compile-darwin:
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags "${LINKER_FLAGS} ${RELEASE_BUILD_LINKER_FLAGS}" -o build/bin/yagss-darwin-amd64 ./cmd/yagss

.PHONY: gen
gen:
	go generate ./...

.PHONY: clean
clean:
	rm -rf build/*
	rm -rf internal/proj/data
