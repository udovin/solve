package logs

import (
	"fmt"
	"runtime"

	"github.com/labstack/gommon/log"
)

type Logger struct {
	*log.Logger
	fields []any
}

func NewLogger() *Logger {
	return &Logger{Logger: log.New("")}
}

func (l *Logger) With(args ...any) *Logger {
	return &Logger{
		Logger: l.Logger,
		fields: append(args, l.fields...),
	}
}

func (l *Logger) Debug(args ...any) {
	log := map[string]any{}
	setLogLine(log, args...)
	l.debugj(log)
}

func (l *Logger) Info(args ...any) {
	log := map[string]any{}
	setLogLine(log, args...)
	l.infoj(log)
}

func (l *Logger) Warn(args ...any) {
	log := map[string]any{}
	setLogLine(log, args...)
	l.warnj(log)
}

func (l *Logger) Error(args ...any) {
	log := map[string]any{}
	setLogLine(log, args...)
	l.errorj(log)
}

func (l *Logger) Fatal(args ...any) {
	log := map[string]any{}
	setLogLine(log, args...)
	l.fatalj(log)
}

func (l *Logger) Debugj(log log.JSON) {
	l.debugj(log)
}

func (l *Logger) Infoj(log log.JSON) {
	l.infoj(log)
}

func (l *Logger) Warnj(log log.JSON) {
	l.warnj(log)
}

func (l *Logger) Errorj(log log.JSON) {
	l.errorj(log)
}

func (l *Logger) Fatalj(log log.JSON) {
	l.fatalj(log)
}

func (l *Logger) Debugf(format string, args ...any) {
	log := map[string]any{}
	setLogLine(log, fmt.Sprintf(format, args...))
	l.debugj(log)
}

func (l *Logger) Infof(format string, args ...any) {
	log := map[string]any{}
	setLogLine(log, fmt.Sprintf(format, args...))
	l.infoj(log)
}

func (l *Logger) Warnf(format string, args ...any) {
	log := map[string]any{}
	setLogLine(log, fmt.Sprintf(format, args...))
	l.warnj(log)
}

func (l *Logger) Errorf(format string, args ...any) {
	log := map[string]any{}
	setLogLine(log, fmt.Sprintf(format, args...))
	l.errorj(log)
}

func (l *Logger) Fatalf(format string, args ...any) {
	log := map[string]any{}
	setLogLine(log, fmt.Sprintf(format, args...))
	l.fatalj(log)
}

func (l *Logger) debugj(log log.JSON) {
	setBaseLogLine(log)
	setLogLine(log, l.fields...)
	l.Logger.Debugj(log)
}

func (l *Logger) infoj(log log.JSON) {
	setBaseLogLine(log)
	setLogLine(log, l.fields...)
	l.Logger.Infoj(log)
}

func (l *Logger) warnj(log log.JSON) {
	setBaseLogLine(log)
	setLogLine(log, l.fields...)
	l.Logger.Warnj(log)
}

func (l *Logger) errorj(log log.JSON) {
	setBaseLogLine(log)
	setLogLine(log, l.fields...)
	l.Logger.Errorj(log)
}

func (l *Logger) fatalj(log log.JSON) {
	setBaseLogLine(log)
	setLogLine(log, l.fields...)
	l.Logger.Fatalj(log)
}

type LogField struct {
	Name  string
	Value any
}

func Any(name string, value any) LogField {
	return LogField{Name: name, Value: value}
}

func setBaseLogLine(log map[string]any) {
	_, file, line, _ := runtime.Caller(3)
	log["file"] = fmt.Sprintf("%s:%d", file, line)
}

func setLogLine(log map[string]any, args ...any) {
	for _, arg := range args {
		switch v := arg.(type) {
		case nil:
		case string:
			log["message"] = v
		case LogField:
			log[v.Name] = v.Value
		case error:
			log["error"] = v.Error()
		default:
			panic(fmt.Errorf("unsupported type: %T", arg))
		}
	}
}
