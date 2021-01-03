.PHONY: clean

yasst:
	mkdir -p build/bin
	go build -o build/bin/yasst cmd/yasst/main.go

clean:
	rm -rf build/*
