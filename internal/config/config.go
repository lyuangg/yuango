package config

import (
	"log"
	"os"
	"reflect"
	"strings"

	"github.com/spf13/viper"
)

const EnvPrefix string = "YUANGO"

// Config holds application runtime configuration.
type Config struct {
	Port     string         `mapstructure:"port" default:"8080"`
	App      AppConfig      `mapstructure:"app"`
	Database DatabaseConfig `mapstructure:"database"`
	Log      LogConfig      `mapstructure:"log"`
}

type AppConfig struct {
	Name string `mapstructure:"name" default:"yuango"`
}

type DatabaseConfig struct {
	Driver          string `mapstructure:"driver" default:"mysql"`
	DSN             string
	Host            string `mapstructure:"host" default:"localhost"`
	Port            int    `mapstructure:"port" default:"3306"`
	User            string `mapstructure:"user" default:"root"`
	Password        string `mapstructure:"password" default:"123456"`
	Name            string `mapstructure:"name" default:"yuango"`
	SSLMode         string `mapstructure:"sslmode" default:"disable"`
	MaxOpenConns    int    `mapstructure:"max_open_conns" default:"50"`
	MaxIdleConns    int    `mapstructure:"max_idle_conns" default:"10"`
	ConnMaxLifetime string `mapstructure:"conn_max_lifetime" default:"1h"` // e.g. 1h, 30m
}

type LogConfig struct {
	Level  string `mapstructure:"level" default:"info"`    // debug, info, warn, error
	Format string `mapstructure:"format" default:"text"`   // text, json
	Output string `mapstructure:"output" default:"stdout"` // stdout, stderr, file path
	Rotate string `mapstructure:"rotate" default:""`       // "daily", "hourly"
}

// setDefaultsFromStruct uses reflection to set default values in Viper from struct tags.
func setDefaultsFromStruct(v *viper.Viper, s interface{}) {
	setDefaultsRecursive(v, reflect.TypeOf(s), "")
}

func setDefaultsRecursive(v *viper.Viper, t reflect.Type, parentKey string) {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		mapstructureTag := field.Tag.Get("mapstructure")
		if mapstructureTag == "" {
			continue
		}

		fullKey := mapstructureTag
		if parentKey != "" {
			fullKey = parentKey + "." + mapstructureTag
		}

		if field.Type.Kind() == reflect.Struct {
			setDefaultsRecursive(v, field.Type, fullKey)
		} else if field.Type.Kind() == reflect.Ptr {
			if field.Type.Elem().Kind() == reflect.Struct {
				setDefaultsRecursive(v, field.Type.Elem(), fullKey)
			} else {
				defaultTag := field.Tag.Get("default")
				if defaultTag != "" {
					v.SetDefault(fullKey, defaultTag)
				}
			}
		} else {
			defaultTag := field.Tag.Get("default")
			if defaultTag != "" {
				v.SetDefault(fullKey, defaultTag)
			}
		}
	}
}

// setupViper configures a viper instance with defaults and environment variable bindings.
func setupViper(v *viper.Viper) {
	// 1. Set defaults from the struct tags.
	setDefaultsFromStruct(v, Config{})

	// 2. Bind environment variables.
	// This allows overriding config values with env vars, e.g., PORT=9090.
	v.AutomaticEnv()

	v.SetEnvPrefix(EnvPrefix)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// We don't need individual BindEnv calls if we use AutomaticEnv and a replacer.
	// For example, `database.max_open_conns` is automatically bound to `DATABASE_MAX_OPEN_CONNS`.
}

// Load reads configuration using viper, with a hierarchy of defaults, config file, and env variables.
func Load() Config {
	v := viper.New()
	setupViper(v)

	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath("./configs")
	v.AddConfigPath(".")
	v.AddConfigPath(os.Getenv(EnvPrefix + "_CONFIG_PATH")) // Allow setting config path via env

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; ignore error and rely on defaults/env
			log.Printf("config: no config file found, using defaults/env: %v", err)
		} else {
			// Config file was found but another error was produced
			log.Fatalf("config: failed to read config: %v", err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		log.Fatalf("config: unmarshal error: %v", err)
	}
	return cfg
}

// LoadFromFile loads configuration from an explicit file path.
// It's stricter than Load() and will exit if the file cannot be read.
func LoadFromFile(path string) Config {
	if path == "" {
		return Load()
	}

	v := viper.New()
	setupViper(v)

	v.SetConfigFile(path)
	v.SetConfigType("yaml")

	if err := v.ReadInConfig(); err != nil {
		log.Fatalf("config: failed to read config file '%s': %v", path, err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		log.Fatalf("config: unmarshal error: %v", err)
	}
	return cfg
}
