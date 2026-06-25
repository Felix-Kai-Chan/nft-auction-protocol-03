// internal/config/config.go
package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	// 数据库
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string

	// 以太坊
	RPCURL       string
	ContractAddr string
	PrivateKey   string
	ChainID      int64

	// 服务
	APIPort string
	Env     string
}

// Load 加载配置
func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️ No .env file found, using environment variables")
	}

	return &Config{
		DBHost:     getEnv("DB_HOST", "127.0.0.1"),
		DBPort:     getEnv("DB_PORT", "3306"),
		DBUser:     getEnv("DB_USER", "root"),
		DBPassword: getEnv("DB_PASSWORD", ""),
		DBName:     getEnv("DB_NAME", "offchain"),

		RPCURL:       getEnv("RPC_URL", "http://localhost:8545"),
		ContractAddr: getEnv("CONTRACT_ADDRESS", ""),
		PrivateKey:   getEnv("PRIVATE_KEY", ""),
		ChainID:      getEnvAsInt("CHAIN_ID", 31337),

		APIPort: getEnv("API_PORT", "8080"),
		Env:     getEnv("ENVIRONMENT", "development"),
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvAsInt(key string, defaultVal int64) int64 {
	if val := os.Getenv(key); val != "" {
		v, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return defaultVal
		}
		return v
	}
	return defaultVal
}
