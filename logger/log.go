/*
 * Copyright (c) 2020 The JaxNetwork developers
 * Use of this source code is governed by an ISC
 * license that can be found in the LICENSE file.
 */

package logger

import (
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/natefinch/lumberjack.v2"
	"io"
	"os"
	"path"
)

var (
	Log *Logger
)

type config struct {
	ConsoleLoggingEnabled bool
	EncodeLogsAsJson      bool

	FileLoggingEnabled bool
	Directory          string
	Filename           string
	MaxSizeMB          int
	MaxBackups         int
	MaxAgeDays         int
}

type Logger struct {
	*zerolog.Logger
}

func Init() {
	Log = configure(config{
		ConsoleLoggingEnabled: true,
		FileLoggingEnabled:    true,

		Directory:  "./",
		Filename:   "operations.log",
		MaxSizeMB:  10,
		MaxBackups: 1,
		MaxAgeDays: 7,
	})

}

// configure sets up the logging framework
//
// In production, the container logs will be collected and file logging should be disabled. However,
// during development it's nicer to see logs as text and optionally write to a file when debugging
// problems in the containerized pipeline
//
// The output log file will be located at /var/log/service-xyz/service-xyz.log and
// will be rolled according to configuration set.
func configure(config config) *Logger {
	var writers []io.Writer

	if config.ConsoleLoggingEnabled {
		writers = append(writers, zerolog.ConsoleWriter{Out: os.Stderr})
	}
	if config.FileLoggingEnabled {
		writers = append(writers, newRollingFile(config))
	}
	mw := io.MultiWriter(writers...)

	zerolog.SetGlobalLevel(zerolog.TraceLevel)
	logger := zerolog.New(mw).With().Timestamp().Logger()

	logger.Info().
		Bool("fileLogging", config.FileLoggingEnabled).
		Bool("jsonLogOutput", config.EncodeLogsAsJson).
		Str("logDirectory", config.Directory).
		Str("fileName", config.Filename).
		Int("maxSizeMB", config.MaxSizeMB).
		Int("maxBackups", config.MaxBackups).
		Int("maxAgeInDays", config.MaxAgeDays).
		Msg("logging configured")

	return &Logger{
		Logger: &logger,
	}
}

func newRollingFile(config config) io.Writer {
	if err := os.MkdirAll(config.Directory, 0744); err != nil {
		log.Error().Err(err).Str("path", config.Directory).Msg("can't create log directory")
		return nil
	}

	return &lumberjack.Logger{
		Filename:   path.Join(config.Directory, config.Filename),
		MaxBackups: config.MaxBackups, // files
		MaxSize:    config.MaxSizeMB,  // megabytes
		MaxAge:     config.MaxAgeDays, // days
	}
}
