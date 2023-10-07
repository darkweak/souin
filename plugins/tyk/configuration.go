package main

import (
	"bytes"
	"sync"

	"github.com/darkweak/souin/context"
	"github.com/darkweak/souin/pkg/middleware"
	"github.com/darkweak/souin/plugins/souin/agnostic"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	configKey       string = "httpcache"
	path            string = "path"
	url             string = "url"
	configurationPK string = "configuration"
)

func parseConfiguration(id string, c map[string]interface{}) *souinInstance {
	c = c[configKey].(map[string]interface{})
	var configuration middleware.BaseConfiguration
	agnostic.ParseConfiguration(&configuration, c)

	s := newInstanceFromConfiguration(configuration)
	definitions[id] = s

	return s
}

func newInstanceFromConfiguration(c middleware.BaseConfiguration) *souinInstance {
	bufPool := &sync.Pool{
		New: func() interface{} {
			return new(bytes.Buffer)
		},
	}

	if c.GetLogger() == nil {
		var logLevel zapcore.Level
		if c.GetLogLevel() == "" {
			logLevel = zapcore.FatalLevel
		} else if err := logLevel.UnmarshalText([]byte(c.GetLogLevel())); err != nil {
			logLevel = zapcore.FatalLevel
		}
		cfg := zap.Config{
			Encoding:         "json",
			Level:            zap.NewAtomicLevelAt(logLevel),
			OutputPaths:      []string{"stderr"},
			ErrorOutputPaths: []string{"stderr"},
			EncoderConfig: zapcore.EncoderConfig{
				MessageKey: "message",

				LevelKey:    "level",
				EncodeLevel: zapcore.CapitalLevelEncoder,

				TimeKey:    "time",
				EncodeTime: zapcore.ISO8601TimeEncoder,

				CallerKey:    "caller",
				EncodeCaller: zapcore.ShortCallerEncoder,
			},
		}
		logger, _ := cfg.Build()
		c.SetLogger(logger)
	}
	ctx := context.GetContext()
	ctx.Init(&c)
	return &souinInstance{
		SouinBaseHandler: middleware.NewHTTPCacheHandler(&c),
		bufPool:          bufPool,
		context:          ctx,
	}
}
