package api

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/labstack/gommon/log"
	"go.uber.org/zap"
)

// ZapWrap represents Zap wrapper for Echo.
type ZapWrap struct {
	*zap.Logger
}

// Output returns nil.
func (z ZapWrap) Output() io.Writer {
	return nil
}

// SetOutput does nothing.
func (z ZapWrap) SetOutput(_ io.Writer) {}

// Prefix returns empty string.
func (z ZapWrap) Prefix() string {
	return ""
}

// SetPrefix does nothing.
func (z ZapWrap) SetPrefix(_ string) {}

// Level returns INFO level.
func (z ZapWrap) Level() log.Lvl {
	return log.INFO
}

// SetLevel does nothing.
func (z ZapWrap) SetLevel(_ log.Lvl) {}

// SetHeader does nothing.
func (z ZapWrap) SetHeader(_ string) {}

// Print prints log line.
func (z ZapWrap) Print(i ...interface{}) {
	z.Info(i...)
}

// Printf prints formatted log line.
func (z ZapWrap) Printf(format string, args ...interface{}) {
	z.Infof(format, args...)
}

// Printj prints JSON line.
func (z ZapWrap) Printj(j log.JSON) {
	b, err := json.Marshal(j)
	if err != nil {
		panic(err)
	}
	z.Print(b)
}

// Debug.
func (z ZapWrap) Debug(i ...interface{}) {
	z.Logger.Debug(fmt.Sprint(i...))
}

// Debugf.
func (z ZapWrap) Debugf(format string, args ...interface{}) {
	z.Logger.Debug(fmt.Sprintf(format, args...))
}

// Debugj.
func (z ZapWrap) Debugj(j log.JSON) {
	b, err := json.Marshal(j)
	if err != nil {
		panic(err)
	}
	z.Debug(b)
}

// Info.
func (z ZapWrap) Info(i ...interface{}) {
	z.Logger.Info(fmt.Sprint(i...))
}

// Infof.
func (z ZapWrap) Infof(format string, args ...interface{}) {
	z.Logger.Info(fmt.Sprintf(format, args...))
}

// Infoj.
func (z ZapWrap) Infoj(j log.JSON) {
	b, err := json.Marshal(j)
	if err != nil {
		panic(err)
	}
	z.Info(b)
}

// Warn.
func (z ZapWrap) Warn(i ...interface{}) {
	z.Logger.Warn(fmt.Sprint(i...))
}

// Warnf.
func (z ZapWrap) Warnf(format string, args ...interface{}) {
	z.Logger.Warn(fmt.Sprintf(format, args...))
}

// Warnj.
func (z ZapWrap) Warnj(j log.JSON) {
	b, err := json.Marshal(j)
	if err != nil {
		panic(err)
	}
	z.Warn(b)
}

// Error.
func (z ZapWrap) Error(i ...interface{}) {
	z.Logger.Error(fmt.Sprint(i...))
}

// Errorf.
func (z ZapWrap) Errorf(format string, args ...interface{}) {
	z.Logger.Error(fmt.Sprintf(format, args...))
}

// Errorj.
func (z ZapWrap) Errorj(j log.JSON) {
	b, err := json.Marshal(j)
	if err != nil {
		panic(err)
	}
	z.Error(b)
}

// Fatal.
func (z ZapWrap) Fatal(i ...interface{}) {
	z.Logger.Fatal(fmt.Sprint(i...))
}

// Fatalj.
func (z ZapWrap) Fatalj(j log.JSON) {
	b, err := json.Marshal(j)
	if err != nil {
		panic(err)
	}
	z.Fatal(b)
}

// Fatalf.
func (z ZapWrap) Fatalf(format string, args ...interface{}) {
	z.Logger.Fatal(fmt.Sprintf(format, args...))
}

// Panic.
func (z ZapWrap) Panic(i ...interface{}) {
	z.Logger.Panic(fmt.Sprint(i...))
}

// Panicf.
func (z ZapWrap) Panicf(format string, args ...interface{}) {
	z.Logger.Panic(fmt.Sprintf(format, args...))
}

// Panicj.
func (z ZapWrap) Panicj(j log.JSON) {
	b, err := json.Marshal(j)
	if err != nil {
		panic(err)
	}
	z.Panic(b)
}
