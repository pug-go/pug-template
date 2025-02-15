package main

import (
	"github.com/pug-go/pug-template/internal/handler"
	"github.com/pug-go/pug-template/internal/server"
)

const grpcPort = 8080
const httpPort = 8090

func main() {
	// TODO: Add config and parsing

	handlers := handler.New()

	grpcServer := server.NewGrpcServer(grpcPort)
	go func() {
		err := grpcServer.Run(handlers.RegisterGrpcServices)
		if err != nil {
			panic(err)
		}
	}()

	httpServer := server.NewHttpServer(httpPort, grpcPort)
	if err := httpServer.Run(handlers.InitHttpRoutes); err != nil {
		panic(err)
	}
}
