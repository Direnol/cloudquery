package logging

import (
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/cloudquery/cloudquery/pkg/ui"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Config for logging
type Config struct {
	// Enable console logging
	ConsoleLoggingEnabled bool `hcl:"enable_console_logging,optional"`
	// Enable Verbose logging
	Verbose bool `hcl:"verbose,optional"`
	// EncodeLogsAsJson makes the logging framework logging JSON
	EncodeLogsAsJson bool `hcl:"encode_logs_as_json,optional"`
	// FileLoggingEnabled makes the framework logging to a file
	// the fields below can be skipped if this value is false!
	FileLoggingEnabled bool `hcl:"file_logging_enabled,optional"`
	// Directory to logging to to when file logging is enabled
	Directory string `hcl:"directory,optional"`
	// Filename is the name of the logfile which will be placed inside the directory
	Filename string `hcl:"filename,optional"`
	// MaxSize the max size in MB of the logfile before it's rolled
	MaxSize int `hcl:"max_size,optional"`
	// MaxBackups the max number of rolled files to keep
	MaxBackups int `hcl:"max_backups,optional"`
	// MaxAge the max age in days to keep a logfile
	MaxAge int `hcl:"max_age,optional"`
	// Console logging will be without color, console logging must be enabled first.
	ConsoleNoColor bool `hcl:"console_no_color,optional"`

	// console is a writer that will be used for console output. If it is not set os.Stderr will be used.
	console io.Writer
}

// Configure sets up the logging framework
//
// In production, the container logs will be collected and file logging should be disabled. However,
// during development it's nicer to see logs as text and optionally write to a file when debugging
// problems in the containerized pipeline
//
// The output logging file should be located at /var/logging/service-xyz/service-xyz.logging and
// will be rolled according to configuration set.
func Configure(config Config) zerolog.Logger {
	var writers []io.Writer

	if config.ConsoleLoggingEnabled {
		if config.EncodeLogsAsJson {
			writers = append(writers, os.Stdout)
		} else {
			console := config.console
			if console == nil {
				console = os.Stderr
			}
			writers = append(writers, zerolog.ConsoleWriter{FormatLevel: formatLevel, Out: console})
		}
	}

	if config.FileLoggingEnabled {
		writers = append(writers, newRollingFile(config))
	}
	mw := io.MultiWriter(writers...)

	// Default level is info, unless verbose flag is on
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if config.Verbose {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	logger := zerolog.New(mw).With().Timestamp().Logger()

	logger.Info().
		Bool("fileLogging", config.FileLoggingEnabled).
		Bool("jsonLogOutput", config.EncodeLogsAsJson).
		Bool("consoleLog", config.ConsoleLoggingEnabled).
		Bool("verbose", config.Verbose).
		Str("logDirectory", config.Directory).
		Str("fileName", config.Filename).
		Int("maxSizeMB", config.MaxSize).
		Int("maxBackups", config.MaxBackups).
		Int("maxAgeInDays", config.MaxAge).
		Msg("logging configured")

	return logger
}

func newRollingFile(config Config) io.Writer {
	if err := os.MkdirAll(config.Directory, 0744); err != nil {
		log.Error().Err(err).Str("path", config.Directory).Msg("can't create logging directory")
		return nil
	}

	return &lumberjack.Logger{
		Filename:   path.Join(config.Directory, config.Filename),
		MaxBackups: config.MaxBackups, // files
		MaxSize:    config.MaxSize,    // megabytes
		MaxAge:     config.MaxAge,     // days
	}
}

// formatLevel is zerolog.Formatter that turns a level value into a string.
func formatLevel(i interface{}) string {
	if level, ok := i.(string); ok {
		switch level {
		case "trace":
			return ui.ColorDebug.Sprint("TRC")
		case "debug":
			return ui.ColorDebug.Sprint("DBG")
		case "info":
			return ui.ColorInfo.Sprint("INF")
		case "warn":
			return ui.ColorWarning.Sprint("WRN")
		case "error":
			return ui.ColorError.Sprint("ERR")
		case "fatal":
			return ui.ColorError.Sprint("FTL")
		case "panic":
			return ui.ColorError.Sprint("PNC")
		default:
			return ui.ColorInfo.Sprint("???")
		}
	}
	if i == nil {
		return ui.ColorInfo.Sprint("???")
	}
	return strings.ToUpper(fmt.Sprintf("%s", i))[0:3]
}
