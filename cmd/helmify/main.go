package main

import (
	"flag"
	"github.com/arttor/helmify/pkg/app"
	"github.com/arttor/helmify/pkg/config"
	"github.com/sirupsen/logrus"
	"os"
)

func main() {
	var chartName string
	flag.StringVar(&chartName, "name", "chart", "chart name")
	flag.Parse()
	stat, err := os.Stdin.Stat()
	if err!=nil {
		logrus.WithError(err).Error("stdin error")
		os.Exit(1)
	}
	if (stat.Mode() & os.ModeCharDevice) != 0 {
		logrus.Warn("no data piped in stdin")
		os.Exit(1)
	}
	if err := app.Start(os.Stdin, config.Config{ChartName: chartName}); err != nil {
		logrus.WithError(err).Error("helmify finished with error")
		os.Exit(1)
	}
}
