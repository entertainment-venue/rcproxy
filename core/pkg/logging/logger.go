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
	"fmt"

	"github.com/sirupsen/logrus"
)

var logObj *logger = nil

func Debug(v ...interface{}) {
	if logObj == nil {
		fmt.Println(append([]interface{}{"[DEBUG]"}, v...)...)
		return
	}
	logObj.iWriter.Debug(v...)
}

func Debugf(format string, v ...interface{}) {
	if logObj == nil {
		fmt.Printf("[DEBUG] "+format+"\n", v...)
		return
	}
	if logObj.iWriter.IsLevelEnabled(logrus.DebugLevel) {
		logObj.iWriter.Debugf(format, v...)
	}
}

// Debugfunc delay string concatenation in func to avoid unnecessary consumption at higher log levels
func Debugfunc(f func() string) {
	if logObj == nil {
		fmt.Print("[DEBUG] " + f() + "\n")
		return
	}
	if logObj.iWriter.IsLevelEnabled(logrus.DebugLevel) {
		logObj.iWriter.Debug(f())
	}
}

func Info(v ...interface{}) {
	if logObj == nil {
		fmt.Println(append([]interface{}{"[INFO]"}, v...)...)
		return
	}
	if logObj.iWriter.IsLevelEnabled(logrus.InfoLevel) {
		logObj.iWriter.Info(v...)
	}
}

func Infof(format string, v ...interface{}) {
	if logObj == nil {
		fmt.Printf("[INFO] "+format+"\n", v...)
		return
	}
	if logObj.iWriter.IsLevelEnabled(logrus.InfoLevel) {
		logObj.iWriter.Infof(format, v...)
	}
}

func Warn(v ...interface{}) {
	if logObj == nil {
		fmt.Println(append([]interface{}{"[WARN]"}, v...)...)
		return
	}
	if logObj.fWriter.IsLevelEnabled(logrus.WarnLevel) {
		logObj.fWriter.Warn(v...)
	}
}

func Warnf(format string, v ...interface{}) {
	if logObj == nil {
		fmt.Printf("[WARN] "+format+"\n", v...)
		return
	}
	if logObj.fWriter.IsLevelEnabled(logrus.WarnLevel) {
		logObj.fWriter.Warnf(format, v...)
	}
}

func Error(v ...interface{}) {
	if logObj == nil {
		fmt.Println(append([]interface{}{"[ERROR]"}, v...)...)
		return
	}
	if logObj.fWriter.IsLevelEnabled(logrus.ErrorLevel) {
		logObj.fWriter.Error(v...)
	}
}

func Errorf(format string, v ...interface{}) {
	if logObj == nil {
		fmt.Printf("[ERROR] "+format+"\n", v...)
		return
	}
	if logObj.fWriter.IsLevelEnabled(logrus.ErrorLevel) {
		logObj.fWriter.Errorf(format, v...)
	}
}
