package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"
)

type Level int

const (
	DEBUG Level = iota
	INFO
	WARN
	ERROR
)

var levelNames = map[Level]string{
	DEBUG: "DEBUG",
	INFO:  "INFO",
	WARN:  "WARN",
	ERROR: "ERROR",
}

type Logger struct {
	mu     sync.Mutex
	out    io.Writer
	level  Level
	prefix string
}

var defaultLogger = &Logger{
	out:   os.Stdout,
	level: INFO,
}

func New(out io.Writer, level Level) *Logger {
	if out == nil {
		out = os.Stdout
	}
	return &Logger{
		out:   out,
		level: level,
	}
}

func SetLevel(level Level) {
	defaultLogger.mu.Lock()
	defaultLogger.level = level
	defaultLogger.mu.Unlock()
}

func SetOutput(w io.Writer) {
	defaultLogger.mu.Lock()
	defaultLogger.out = w
	defaultLogger.mu.Unlock()
}

func (l *Logger) log(level Level, format string, v ...interface{}) {
	if level < l.level {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	levelStr := levelNames[level]
	msg := fmt.Sprintf(format, v...)
	
	line := fmt.Sprintf("[%s] %s: %s\n", timestamp, levelStr, msg)
	l.out.Write([]byte(line))
}

func (l *Logger) Debug(format string, v ...interface{}) {
	l.log(DEBUG, format, v...)
}

func (l *Logger) Info(format string, v ...interface{}) {
	l.log(INFO, format, v...)
}

func (l *Logger) Warn(format string, v ...interface{}) {
	l.log(WARN, format, v...)
}

func (l *Logger) Error(format string, v ...interface{}) {
	l.log(ERROR, format, v...)
}

// Global functions
func Debug(format string, v ...interface{}) {
	defaultLogger.log(DEBUG, format, v...)
}

func Info(format string, v ...interface{}) {
	defaultLogger.log(INFO, format, v...)
}

func Warn(format string, v ...interface{}) {
	defaultLogger.log(WARN, format, v...)
}

func Error(format string, v ...interface{}) {
	defaultLogger.log(ERROR, format, v...)
}

// Compatibility with standard log
func Printf(format string, v ...interface{}) {
	defaultLogger.log(INFO, format, v...)
}

func Init() {
	log.SetOutput(defaultLogger.out)
	log.SetFlags(0)
}
