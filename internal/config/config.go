package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/joho/godotenv"
)

type Contract struct {
	Address common.Address
	ABI     abi.ABI
}

type DbConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DbName   string
}

type Config struct {
	Environment string
	API_PORT    int
	Db          DbConfig
	AdminAPIKey string

	RPC_URL       string
	Web3SignerURL string
	BGTContract   Contract
	GasLimit      int

	CronSchedule string
}

func LoadConfig() *Config {
	if err := LoadEnv(); err != nil {
		panic(fmt.Sprintf("Error loading environment variables: %v", err))
	}

	bgtContract := getEnvString("BGT_CONTRACT", ptr("0x656b95E550C07a9ffe548bd4085c72418Ceb1dba"))
	bgtABI, err := readABI("abi.json")
	if err != nil {
		panic(fmt.Sprintf("Error reading ABI: %v", err))
	}

	config := Config{
		Environment: getEnvString("ENVIRONMENT", ptr("development")),
		API_PORT:    getEnvInt("API_PORT", ptr(8080)),
		Db: DbConfig{
			Host:     getEnvString("DB_HOST", ptr("localhost")),
			User:     getEnvString("DB_USER", ptr("")),
			Password: getEnvString("DB_PASS", ptr("")),
			DbName:   getEnvString("DB_NAME", ptr("bgt_boost")),
			Port:     getEnvInt("DB_PORT", ptr(27017)),
		},
		AdminAPIKey: getEnvString("ADMIN_API_KEY", nil),

		RPC_URL:       getEnvString("RPC_URL", nil),
		Web3SignerURL: getEnvString("WEB3SIGNER_URL", nil),
		BGTContract: Contract{
			Address: common.HexToAddress(bgtContract),
			ABI:     bgtABI,
		},
		GasLimit: getEnvInt("GAS_LIMIT", ptr(150000)),

		CronSchedule: getEnvString("CRON_SCHEDULE", ptr("0 */5 * * * *")),
	}
	log.Println("✅ Config Loaded")
	return &config
}

func getConfigPath() (string, error) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("error getting current file path")
	}
	return filepath.Dir(filename), nil
}

func LoadEnv() error {
	dir, err := getConfigPath()
	if err != nil {
		return err
	}

	envPath := filepath.Join(dir, "../../.env")
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		// .env file doesn't exist, just return without an error
		return nil
	}

	return godotenv.Load(envPath)
}

func readABI(filePath string) (abi.ABI, error) {
	dir, err := getConfigPath()
	if err != nil {
		return abi.ABI{}, err
	}

	abiPath := filepath.Join(dir, filePath)
	abiFile, err := os.ReadFile(abiPath)
	if err != nil {
		return abi.ABI{}, fmt.Errorf("failed to read ABI file: %v", err)
	}

	contractABI, err := abi.JSON(strings.NewReader(string(abiFile)))
	if err != nil {
		return abi.ABI{}, fmt.Errorf("failed to parse ABI: %v", err)
	}
	return contractABI, nil
}

func getEnvString(key string, defaultValue *string) string {
	value := os.Getenv(key)

	if value != "" {
		return value
	}
	if defaultValue == nil {
		panic(fmt.Sprintf("Environment variable %s is required", key))
	}
	return *defaultValue
}

func getEnvInt(key string, defaultValue *int) int {
	value := os.Getenv(key)
	if value != "" {
		intValue, err := strconv.Atoi(value)
		if err != nil {
			panic(fmt.Sprintf("Environment variable %s is not a valid integer", key))
		}
		return intValue
	}
	if defaultValue == nil {
		panic(fmt.Sprintf("Environment variable %s is required", key))
	}
	return *defaultValue
}

func ptr[T any](v T) *T {
	return &v
}
