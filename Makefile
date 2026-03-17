.PHONY: build test proto clean

build:
	go build -o jess-npm ./cmd/jess-npm

test:
	go test ./... -v -race

proto:
	protoc \
		--proto_path=. \
		--proto_path=../../_protos \
		--go_out=gen/go \
		'--go_opt=paths=source_relative' \
		'--go_opt=Mnpm/v1/npm.proto=github.com/organic-programming/jess-npm/gen/go/npm/v1;npmv1' \
		--go-grpc_out=gen/go \
		'--go-grpc_opt=paths=source_relative' \
		'--go-grpc_opt=Mnpm/v1/npm.proto=github.com/organic-programming/jess-npm/gen/go/npm/v1;npmv1' \
		npm/v1/npm.proto

clean:
	rm -f jess-npm
	go clean -cache
