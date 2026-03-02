.PHONY: build test proto clean

build:
	go build -o jess-npm ./cmd/jess-npm

test:
	go test ./... -v -race

proto:
	protoc \
		--go_out=. --go_opt=module=github.com/organic-programming/jess-npm \
		--go-grpc_out=. --go-grpc_opt=module=github.com/organic-programming/jess-npm \
		protos/npm/v1/npm.proto

clean:
	rm -f jess-npm
	go clean -cache
