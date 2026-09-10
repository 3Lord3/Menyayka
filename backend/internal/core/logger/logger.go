package logger

import "github.com/sirupsen/logrus"

// New creates the application logger with the given level.
// An invalid or empty level falls back to info. It also sets the standard
// logrus logger level so package-level calls use the same level as the app.
func New(level string) *logrus.Logger {
	l := logrus.New()

	lvl, err := logrus.ParseLevel(level)
	if err != nil {
		lvl = logrus.InfoLevel
	}
	l.SetLevel(lvl)
	logrus.SetLevel(lvl)

	return l
}
