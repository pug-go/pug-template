package main

import (
	"flag"

	"github.com/pug-go/pug-template/internal/config"
	"github.com/pug-go/pug-template/internal/handler"
	"github.com/pug-go/pug-template/internal/server"
)

var flagconf string

func init() {
	flag.StringVar(&flagconf, "conf", "", "config path, example: -conf .deployment/values_local.yaml")
}

func main() {
	flag.Parse()

	cfg := &config.GlobalConfig
	err := cfg.Load(flagconf)
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
