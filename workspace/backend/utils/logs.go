package utils

import (
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/sirupsen/logrus"
)

func ShowErrorLogs(errData error) {
	if errData != nil {
		file, line := Caller(2)
		logrus.SetFormatter(&logrus.TextFormatter{})
		logrus.Errorf("%s[%d] %s", filepath.Base(file), line, errData.Error())
	}
}

func ShowInfoLogs(format string, a ...any) {
	file, line := Caller(2)
	logrus.SetFormatter(&logrus.TextFormatter{})
	if len(a) == 0 {
		logrus.Infof("%s[%d] %s", filepath.Base(file), line, format)
	} else {
		logrus.Infof("%s[%d] %s", filepath.Base(file), line, fmt.Sprintf(format, a...))
	}
}

func Caller(level int) (string, int) {
	_, file, line, _ := runtime.Caller(level)
	return file, line
}
