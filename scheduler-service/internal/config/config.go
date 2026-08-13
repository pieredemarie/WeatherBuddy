package config

import (
	"log"
	"os"
	"strings"
)

type Config struct {
	PostgresDSN  string
	KafkaBrokers []string
}

func MustLoad() *Config {
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		log.Fatal("postgres is not set")
	}

	brokers := os.Getenv("KAFKA_BROKERS")
	if brokers == "" {
		brokers = "localhost:9092"
	}

	return &Config{
		PostgresDSN:  dsn,
		KafkaBrokers: strings.Split(brokers, ","),
	}
}
