package main

import (
	"flag"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/pug-go/pug-template/internal/config"
	"github.com/pug-go/pug-template/internal/handler"
	"github.com/pug-go/pug-template/internal/server"
	"github.com/pug-go/pug-template/pkg/pug"
)

var flagconf string

func init() {
	flag.StringVar(&flagconf, "conf", "", "config path, example: -conf .deployment/values_local.yaml")
}

func main() {
	flag.Parse()

	log.SetFormatter(&log.JSONFormatter{})
	log.SetReportCaller(true)
	time.Local = time.UTC

	cfg := &config.GlobalConfig
	err := cfg.Load(flagconf)
	if err != nil {
		panic(err)
	}

	handlers := handler.New()
	grpcServer := server.NewGrpcServer(handlers.RegisterGrpcServices)
	httpServer := server.NewHttpServer(handlers.InitHttpRoutes)

	app, err := pug.NewApp(pug.Config{
		ServiceName: cfg.Service.Name,
		Domain:      cfg.Service.Domain,
		GrpcPort:    cfg.Service.Ports.Grpc,
		HttpPort:    cfg.Service.Ports.Http,
		DebugPort:   cfg.Service.Ports.Debug,
	})
	if err != nil {
		panic(err)
	}

	app.Run(grpcServer, httpServer)
}
