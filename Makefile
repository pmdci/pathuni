.PHONY: build clean test install

build:
	mkdir -p bin
	go build -o bin/pathuni ./cmd/pathuni

clean:
	rm -rf bin/

test:
	go test ./...

install: build
	cp bin/pathuni $(HOME)/.local/bin/

dev: build
	./bin/pathuni --eval