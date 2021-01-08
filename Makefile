yagss: gen
	mkdir -p build/bin
	go build -o build/bin/yasst cmd/yasst/main.go

.PHONY: gen
gen:
	go generate ./...

.PHONY: clean
clean:
	rm -rf build/*
	rm -rf internal/proj/data
