package config

import (
	"flag"
	"log"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
)

type Config struct {
	Name           string      `yaml:"name"`
	DB             DBConfig    `yaml:"db"`
	Kafka          KafkaConfig `yaml:"kafka"`
	CacheConfig    CacheConfig `yaml:"cache"`
	OutputSource   string      `yaml:"output_source"`
	GRPCPort       int         `yaml:"grpc_port"`
	HTTPPort       int         `yaml:"http_port"`
	PrometheusPort int         `yaml:"prometheus_port"`
}

type CacheConfig struct {
	Type     EvictionStrategy `yaml:"type"`
	TTL      time.Duration    `yaml:"ttl"`
	Capacity int              `yaml:"capacity"`
}

type DBConfig struct {
	Username string `yaml:"username"`
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	DBName   string `yaml:"db_name"`
	SSLMode  string `yaml:"ssl_mode" env-default:"disable"`
	Password string
}

type KafkaConfig struct {
	Brokers []string
	Topic   string
}

func MustLoad() *Config {
	if err := godotenv.Load(); err != nil {
		log.Fatalf("error loading .env file, %s", err)
	}

	configPath := fetchConfigPath()

	cfg := MustLoadPath(configPath)

	cfg.DB.Password = GetValue("DB_PASSWORD", "")

	return cfg
}

func MustLoadPath(configPath string) *Config {
	// check if file exists
	if _, err := os.Stat(configPath); err != nil {
		log.Fatalf("config file does not exist: %s", configPath)
	}

	var cfg Config
	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		log.Fatalf("cannot read config: %s", err)
	}

	return &cfg
}

func fetchConfigPath() string {
	var res string

	flag.StringVar(&res, "config", "", "path to config file")
	flag.Parse()

	if res == "" {
		res = GetValue("CONFIG_PATH", "./config.yml")
	}

	return res
}
