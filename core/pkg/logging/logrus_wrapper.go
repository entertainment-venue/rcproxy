// Copyright (c) 2022 The rcproxy Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package logging

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/lestrrat-go/file-rotatelogs"
	"github.com/sirupsen/logrus"

	"rcproxy/core/pkg/constant"
)

const (
	defaultMaxLength = 8192
)

const (
	LevelDebug = "DEBUG"
	LevelInfo  = "INFO"
	LevelWarn  = "WARN"
	LevelError = "ERROR"
)

var LevelMapperRev = map[string]logrus.Level{
	LevelDebug: logrus.DebugLevel,
	LevelInfo:  logrus.InfoLevel,
	LevelWarn:  logrus.WarnLevel,
	LevelError: logrus.ErrorLevel,
}

type logger struct {
	iWriter *logrus.Logger
	fWriter *logrus.Logger
}

type logOptions struct {
	path      string
	level     string
	expireDay int
}

var defaultLogOptions = logOptions{
	path:      "log",
	level:     LevelDebug,
	expireDay: 7,
}

type logOptionsFunc func(*logOptions)

func WithPath(v string) logOptionsFunc {
	return func(o *logOptions) {
		o.path = v
	}
}

func WithExpireDay(v int) logOptionsFunc {
	return func(o *logOptions) {
		o.expireDay = v
	}
}

func WithLogLevel(l string) logOptionsFunc {
	return func(o *logOptions) {
		o.level = l
	}
}

func InitializeLogger(opt ...logOptionsFunc) error {
	if logObj != nil {
		fmt.Printf("[logging] logObj is already initialized\n")
		return nil
	}
	opts := defaultLogOptions
	for _, o := range opt {
		o(&opts)
	}

	if err := os.MkdirAll(opts.path, os.FileMode(0755)); err != nil {
		fmt.Printf("[logging] mkdir failed, path: %s\n", opts.path)
		return err
	}

	iWriter, err := newWriter(opts.path, "rcproxy.log", opts.expireDay)
	if err != nil {
		return err
	}

	fWriter, err := newWriter(opts.path, "rcproxy.log.wf", opts.expireDay)
	if err != nil {
		return err
	}

	logObj = &logger{
		iWriter: iWriter,
		fWriter: fWriter,
	}
	if v, ok := LevelMapperRev[opts.level]; ok {
		logObj.iWriter.SetLevel(v)
		logObj.fWriter.SetLevel(v)
	}
	return nil
}

func newWriter(filepath, fileName string, expireDay int) (logger *logrus.Logger, err error) {
	var fileWithFullPath string
	if strings.HasPrefix(filepath, "/") {
		fileWithFullPath = path.Join(filepath, fileName)
	} else {
		pwd, err := os.Getwd()
		if err != nil {
			fmt.Printf("[logging] os.getwd err, err: %s\n", err)
			return nil, err
		}
		fileWithFullPath = path.Join(pwd, filepath, fileName)
	}
	logger = logrus.New()
	writer, err := rotatelogs.New(
		fileWithFullPath+".%Y%m%d%H",
		rotatelogs.WithLinkName(fileWithFullPath),
		rotatelogs.WithMaxAge(time.Duration(expireDay)*24*time.Hour),
		rotatelogs.WithRotationTime(time.Hour),
	)
	if err != nil {
		fmt.Printf("[logging] failed to create rotatelogs: %s\n", err)
		return nil, err
	}
	logger.SetOutput(writer)
	logger.Formatter = &textFormatter{}
	return
}

type textFormatter struct{}

func (f *textFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	var b *bytes.Buffer
	message := strings.TrimSuffix(entry.Message, "\n")

	if len(entry.Message) > defaultMaxLength {
		entry.Message = entry.Message[:defaultMaxLength]
	}

	if entry.Buffer != nil {
		b = entry.Buffer
	} else {
		b = &bytes.Buffer{}
	}

	f.appendValue(b, strings.ToUpper(entry.Level.String()))
	b.WriteByte(' ')
	f.appendValue(b, entry.Time.Format("06-01-02 15:04:05.999"))
	b.WriteByte(' ')

	if strings.HasPrefix(message, constant.TitleSlowLog) {
		f.appendValue(b, message)
		b.WriteByte('\n')
		return b.Bytes(), nil
	}

	callers := getCaller(entry.Level)
	if len(callers) > 0 {
		f.appendValue(b, strings.TrimPrefix(callers[0].Function, "rcproxy/"))
		b.WriteByte(' ')
		f.appendValue(b, fmt.Sprintf("%s:%d", filepath.Base(callers[0].File), callers[0].Line))
		b.WriteByte(' ')
	}

	f.appendValue(b, message)
	b.WriteByte('\n')

	if len(callers) > 1 {
		for _, c := range callers {
			b.WriteString("        ")
			f.appendValue(b, strings.TrimPrefix(c.Function, "rcproxy/"))
			b.WriteByte(' ')
			f.appendValue(b, fmt.Sprintf("%s:%d", filepath.Base(c.File), c.Line))
			b.WriteByte('\n')
		}
	}
	return b.Bytes(), nil
}

func (f *textFormatter) appendValue(b *bytes.Buffer, value interface{}) {
	stringVal, ok := value.(string)
	if !ok {
		stringVal = fmt.Sprint(value)
	}
	b.WriteString(stringVal)
}

func getCaller(level logrus.Level) (fms []runtime.Frame) {
	var getPackageName = func(f string) string {
		for {
			lastPeriod := strings.LastIndex(f, ".")
			lastSlash := strings.LastIndex(f, "/")
			if lastPeriod > lastSlash {
				f = f[:lastPeriod]
			} else {
				break
			}
		}
		return f
	}

	pcs := make([]uintptr, 25)
	depth := runtime.Callers(1, pcs)
	frames := runtime.CallersFrames(pcs[:depth])

	for f, again := frames.Next(); again; f, again = frames.Next() {
		pkg := getPackageName(f.Function)
		if strings.Contains(pkg, "rcproxy/core/pkg/log") || strings.Contains(pkg, "sirupsen/logrus") {
			continue
		}
		fms = append(fms, f)
		if level != logrus.ErrorLevel {
			return fms
		}
	}
	return fms
}
