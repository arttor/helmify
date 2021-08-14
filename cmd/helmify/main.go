package main

import (
	"github.com/arttor/helmify/pkg/app"
	"github.com/arttor/helmify/pkg/flags"
	"github.com/sirupsen/logrus"
	"os"
)

func main() {
	conf := flags.Read()
	stat, err := os.Stdin.Stat()
	if err != nil {
		logrus.WithError(err).Error("stdin error")
		os.Exit(1)
	}
	if (stat.Mode() & os.ModeCharDevice) != 0 {
		logrus.Error("no data piped in stdin")
		os.Exit(1)
	}
	if err = app.Start(os.Stdin, conf); err != nil {
		logrus.WithError(err).Error("helmify finished with error")
		os.Exit(1)
	}
}
