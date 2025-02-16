package config

type Config struct {
	GrpcPort  int16 `yaml:"service.ports.grpc" env:"GRPC_PORT" env-default:"8080"`
	HttpPort  int16 `yaml:"service.ports.http" env:"HTTP_PORT" env-default:"8082"`
	DebugPort int16 `yaml:"service.ports.debug" env:"DEBUG_PORT" env-default:"8084"`
}

var GlobalConfig Config
