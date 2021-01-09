yagss: gen
	mkdir -p build/bin
	go build -o build/bin/yagss cmd/yagss/main.go

.PHONY: gen
gen:
	go generate ./...

.PHONY: clean
clean:
	rm -rf build/*
	rm -rf internal/proj/data
