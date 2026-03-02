package main

import (
	"fmt"
	"os"

	"github.com/organic-programming/go-holons/pkg/serve"
	npmv1 "github.com/organic-programming/jess-npm/gen/go/npm/v1"
	"github.com/organic-programming/jess-npm/internal/service"
	"google.golang.org/grpc"
)

func main() {
	listenURI := serve.ParseFlags(os.Args[1:])
	if err := serve.Run(listenURI, func(s *grpc.Server) {
		npmv1.RegisterNpmServiceServer(s, &service.NpmServer{})
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
