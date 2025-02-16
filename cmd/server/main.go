package main

import (
	"github.com/ilyakaznacheev/cleanenv"
	"github.com/pug-go/pug-template/internal/config"
	"github.com/pug-go/pug-template/internal/handler"
	"github.com/pug-go/pug-template/internal/server"
)

func main() {
	cfg := config.GlobalConfig
	err := cleanenv.ReadConfig(".deployment/values_local.yaml", &cfg)
	if err != nil {
		panic(err)
	}

	handlers := handler.New()

	grpcServer := server.NewGrpcServer(cfg.GrpcPort)
	go func() {
		err = grpcServer.Run(handlers.RegisterGrpcServices)
		if err != nil {
			panic(err)
		}
	}()

	httpServer := server.NewHttpServer(cfg.HttpPort, cfg.GrpcPort)
	if err = httpServer.Run(handlers.InitHttpRoutes); err != nil {
		panic(err)
	}
}
