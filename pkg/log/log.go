package log

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"runtime"
	"strings"
)

func init() {
	logrus.SetReportCaller(true)
	formatter := &logrus.TextFormatter{
		TimestampFormat:        "2006-01-02 15:04:05", // the "time" field configuratiom
		FullTimestamp:          true,
		DisableLevelTruncation: true, // log level field configuration
		CallerPrettyfier: func(f *runtime.Frame) (string, string) {
			return "", fmt.Sprintf(" %s:%d", formatFilePath(f.File), f.Line)
		},
	}
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(formatter)
}
func formatFilePath(path string) string {
	arr := strings.Split(path, "/")
	return arr[len(arr)-1]
}
