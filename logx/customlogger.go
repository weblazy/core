package logx

import (
	"fmt"
	"io"
	"time"
)

const customCallerDepth = 3

type customLog struct {
	logEntry
}

func WithDuration(d time.Duration) Logger {
	logger := new(customLog)
	logger.Duration = d.String()
	return logger
}

func (l *customLog) Error(v ...interface{}) {
	if shouldLog(ErrorLevel) {
		l.write(errorLog, levelError, formatWithCaller(fmt.Sprint(v...), customCallerDepth))
	}
}

func (l *customLog) Errorf(format string, v ...interface{}) {
	if shouldLog(ErrorLevel) {
		l.write(errorLog, levelError, formatWithCaller(fmt.Sprintf(format, v...), customCallerDepth))
	}
}

func (l *customLog) Debug(v ...interface{}) {
	if shouldLog(DebugLevel) {
		l.write(debugLog, levelDebug, formatWithCaller(fmt.Sprint(v...), customCallerDepth))
	}
}

func (l *customLog) Debugf(format string, v ...interface{}) {
	if shouldLog(DebugLevel) {
		l.write(debugLog, levelDebug, formatWithCaller(fmt.Sprintf(format, v...), customCallerDepth))
	}
}

func (l *customLog) Info(v ...interface{}) {
	if shouldLog(InfoLevel) {
		l.write(infoLog, levelInfo, fmt.Sprint(v...))
	}
}

func (l *customLog) Infof(format string, v ...interface{}) {
	if shouldLog(InfoLevel) {
		l.write(infoLog, levelInfo, fmt.Sprintf(format, v...))
	}
}

func (l *customLog) Slow(v ...interface{}) {
	if shouldLog(ErrorLevel) {
		l.write(slowLog, levelSlow, fmt.Sprint(v...))
	}
}

func (l *customLog) Slowf(format string, v ...interface{}) {
	if shouldLog(ErrorLevel) {
		l.write(slowLog, levelSlow, fmt.Sprintf(format, v...))
	}
}

func (l *customLog) write(writer io.Writer, level, content string) {
	l.Timestamp = getTimestamp()
	l.Level = level
	l.Content = content
	outputJson(writer, l.logEntry)
}
