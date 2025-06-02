package config

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

const (
	defaultFileName         = "/.env"
	defaultOverrideFileName = "/.local.env"
)

type EnvLoader struct {
}

func NewEnvFile(configFolder string) *AppConfig {
	conf := &EnvLoader{}
	conf.read(configFolder)

	return &AppConfig{}
}

func (e *EnvLoader) read(folder string) {
	var (
		defaultFile  = folder + defaultFileName
		overrideFile = folder + defaultOverrideFileName
		env          = e.Get("APP_ENV")
	)

	err := godotenv.Load(defaultFile)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			log.Fatalf("Failed to load config from file: %v, Err: %v", defaultFile, err)
		}

		log.Printf("Failed to load config from file: %v, Err: %v", defaultFile, err)
	} else {
		log.Printf("Loaded config from file: %v", defaultFile)
	}

	if env != "" {
		// If 'APP_ENV' is set to x, then GoFr will read '.env' from configs directory, and then it will be overwritten
		// by configs present in file '.x.env'
		overrideFile = fmt.Sprintf("%s/.%s.env", folder, env)
	}

	err = godotenv.Overload(overrideFile)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			log.Fatalf("Failed to load config from file: %v, Err: %v", overrideFile, err)
		}
	} else {
		log.Printf("Loaded config from file: %v", overrideFile)
	}

	// Reload system environment variables to ensure they override any previously loaded values
	for _, envVar := range os.Environ() {
		key, value, found := strings.Cut(envVar, "=")
		if found {
			os.Setenv(key, value)
		}
	}
}

func (*EnvLoader) Get(key string) string {
	return os.Getenv(key)
}

func (*EnvLoader) GetOrDefault(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}

	return defaultValue
}
