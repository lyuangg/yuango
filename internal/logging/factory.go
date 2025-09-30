package logging

import (
	"fmt"
	"reflect"
	"strings"
)

// NewFromConfig creates a new logger from configuration.
func NewFromConfig(c interface{}) (Logger, error) {
	var level, format, output string
	var daily bool
	var maxFiles int

	// 处理不同类型的配置
	switch conf := c.(type) {
	case *Config:
		level = conf.Level
		format = conf.Format
		output = conf.Output
		daily = conf.Daily
		maxFiles = conf.MaxFiles
	default:
		// 尝试从旧配置中获取值
		v := reflect.ValueOf(c)
		if v.Kind() == reflect.Struct {
			if l := v.FieldByName("Level"); l.IsValid() && l.Kind() == reflect.String {
				level = l.String()
			}
			if f := v.FieldByName("Format"); f.IsValid() && f.Kind() == reflect.String {
				format = f.String()
			}
			if o := v.FieldByName("Output"); o.IsValid() && o.Kind() == reflect.String {
				output = o.String()
			}
			if d := v.FieldByName("Daily"); d.IsValid() && d.Kind() == reflect.Bool {
				daily = d.Bool()
			}
		}
	}

	logLevel, err := parseLevel(level)
	if err != nil {
		return nil, err
	}

	return NewSlogLogger(logLevel, format, output, daily, maxFiles)
}

func parseLevel(s string) (Level, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return LevelDebug, nil
	case "info", "":
		return LevelInfo, nil
	case "warn", "warning":
		return LevelWarn, nil
	case "error", "err":
		return LevelError, nil
	default:
		return LevelInfo, fmt.Errorf("unknown log level: %s", s)
	}
}
