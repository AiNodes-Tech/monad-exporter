.PHONY: build

build:
	mkdir -p bin
	cd src && go build -buildvcs=false -o ../bin/monad-exporter ./cmd/monad-exporter
