package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/arttor/helmify/pkg/app"
	"github.com/arttor/helmify/pkg/translator/k8smanifest"
	"github.com/sirupsen/logrus"
)

func main() {
	conf, err := ReadFlags()
	if err != nil {
		fmt.Println(err)
		flag.Usage()
		os.Exit(1)
	}
	stat, err := os.Stdin.Stat()
	if err != nil {
		logrus.WithError(err).Error("stdin error")
		os.Exit(1)
	}
	if len(conf.Files) == 0 && (stat.Mode()&os.ModeCharDevice) != 0 {
		logrus.Error("no data piped in stdin")
		os.Exit(1)
	}

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-done
		logrus.Debug("Received termination, signaling shutdown")
		cancelFunc()
	}()

	trans := k8smanifest.New(conf, os.Stdin)
	engine := app.NewEngine(conf)

	if err = engine.Run(ctx, trans); err != nil {
		logrus.WithError(err).Error("helmify finished with error")
		os.Exit(1)
	}
}
