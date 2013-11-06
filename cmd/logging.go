// Copyright 2012, 2013 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"launchpad.net/gnuflag"
	"launchpad.net/loggo"

	"launchpad.net/juju-core/juju/osenv"
)

var (
	logger        = loggo.GetLogger("juju.cmd")
	infoWriter    io.Writer
	verboseWriter io.Writer
)

func writeInternal(writer io.Writer, format string, params ...interface{}) {
	if writer == nil {
		logger.Infof(format, params...)
	} else {
		output := fmt.Sprintf(format, params...)
		if !strings.HasSuffix(output, "\n") {
			output = output + "\n"
		}
		io.WriteString(writer, output)
	}
}

// Infof will write the formatted string to the infoWriter if one has been
// specified, or to the package logger if it hasn't.
func Infof(format string, params ...interface{}) {
	writeInternal(infoWriter, format, params...)
}

// Verbosef will write the formatted string to the verboseWriter if one has been
// specified, or to the package logger if it hasn't.
func Verbosef(format string, params ...interface{}) {
	writeInternal(verboseWriter, format, params...)
}

// Log supplies the necessary functionality for Commands that wish to set up
// logging.
type Log struct {
	Path    string
	Verbose bool
	Quiet   bool
	Debug   bool
	ShowLog bool
	Config  string
}

// AddFlags adds appropriate flags to f.
func (l *Log) AddFlags(f *gnuflag.FlagSet) {
	f.StringVar(&l.Path, "log-file", "", "path to write log to")
	f.BoolVar(&l.Verbose, "v", false, "show more verbose output")
	f.BoolVar(&l.Verbose, "verbose", false, "show more verbose output")
	f.BoolVar(&l.Quiet, "q", false, "show no informational output")
	f.BoolVar(&l.Quiet, "quiet", false, "show no informational output")
	f.BoolVar(&l.Debug, "d", false, "equivalent to --show-log --log-config=<root>=DEBUG")
	f.BoolVar(&l.Debug, "debug", false, "equivalent to --show-log --log-config=<root>=DEBUG")
	defaultLogConfig := os.Getenv(osenv.JujuLoggingConfig)
	f.StringVar(&l.Config, "log-config", defaultLogConfig, "specify log levels for modules")
	f.BoolVar(&l.ShowLog, "show-log", false, "if set, write the log file to stderr")
}

// Start starts logging using the given Context.
func (log *Log) Start(ctx *Context) error {
	if log.Verbose && log.Quiet {
		return fmt.Errorf(`"verbose" and "quiet" flags clash`)
	}
	if !log.Quiet {
		infoWriter = ctx.Stderr
		if log.Verbose {
			verboseWriter = ctx.Stderr
		}
	}
	if log.Path != "" {
		path := ctx.AbsPath(log.Path)
		target, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
		if err != nil {
			return err
		}
		writer := loggo.NewSimpleWriter(target, &loggo.DefaultFormatter{})
		err = loggo.RegisterWriter("logfile", writer, loggo.TRACE)
		if err != nil {
			return err
		}
	}
	level := loggo.WARNING
	if log.ShowLog {
		level = loggo.INFO
	}
	if log.Debug {
		log.ShowLog = true
		level = loggo.DEBUG
	}

	if log.ShowLog {
		// We replace the default writer to use ctx.Stderr rather than os.Stderr.
		writer := loggo.NewSimpleWriter(ctx.Stderr, &loggo.DefaultFormatter{})
		_, err := loggo.ReplaceDefaultWriter(writer)
		if err != nil {
			return err
		}
	} else {
		loggo.RemoveWriter("default")
		// Create a simple writer that doesn't show filenames, or timestamps,
		// and only shows warning or above.
		writer := loggo.NewSimpleWriter(ctx.Stderr, &warningFormatter{})
		err := loggo.RegisterWriter("warning", writer, loggo.WARNING)
		if err != nil {
			return err
		}
	}
	// Set the level on the root logger.
	loggo.GetLogger("").SetLogLevel(level)
	// Override the logging config with specified logging config.
	loggo.ConfigureLoggers(log.Config)
	return nil
}

// warningFormatter is a simple loggo formatter that produces something like:
//   WARNING The message...
type warningFormatter struct{}

func (*warningFormatter) Format(level loggo.Level, _, _ string, _ int, _ time.Time, message string) string {
	return fmt.Sprintf("%s %s", level, message)
}

// NewCommandLogWriter creates a loggo writer for registration
// by the callers of a command. This way the logged output can also
// be displayed otherwise, e.g. on the screen.
func NewCommandLogWriter(name string, out, err io.Writer) loggo.Writer {
	return &commandLogWriter{name, out, err}
}

// commandLogWriter filters the log messages for name.
type commandLogWriter struct {
	name string
	out  io.Writer
	err  io.Writer
}

// Write implements loggo's Writer interface.
func (s *commandLogWriter) Write(level loggo.Level, name, filename string, line int, timestamp time.Time, message string) {
	if name == s.name {
		if level <= loggo.INFO {
			fmt.Fprintf(s.out, "%s\n", message)
		} else {
			fmt.Fprintf(s.err, "%s\n", message)
		}
	}
}
