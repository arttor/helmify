package app

import (
	"context"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"

	"github.com/arttor/helmify/pkg/config"
	"github.com/arttor/helmify/pkg/decoder"
	"github.com/arttor/helmify/pkg/helm"
	"github.com/arttor/helmify/pkg/processor"
	"github.com/arttor/helmify/pkg/processor/configmap"
	"github.com/arttor/helmify/pkg/processor/crd"
	"github.com/arttor/helmify/pkg/processor/deployment"
	"github.com/arttor/helmify/pkg/processor/statefulset"
	"github.com/arttor/helmify/pkg/processor/rbac"
	"github.com/arttor/helmify/pkg/processor/secret"
	"github.com/arttor/helmify/pkg/processor/service"
	"github.com/arttor/helmify/pkg/processor/storage"
	"github.com/arttor/helmify/pkg/processor/webhook"
)

// Start - application entrypoint for processing input to a Helm chart.
func Start(input io.Reader, config config.Config) error {
	err := config.Validate()
	if err != nil {
		return err
	}
	setLogLevel(config)
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-done
		logrus.Debug("Received termination, signaling shutdown")
		cancelFunc()
	}()
	objects := decoder.Decode(ctx.Done(), input)
	appCtx := New(config, helm.NewOutput())
	appCtx = appCtx.WithProcessors(
		configmap.New(),
		crd.New(),
		deployment.New(),
		statefulset.New(),
		storage.New(),
		service.New(),
		service.NewIngress(),
		rbac.ClusterRoleBinding(),
		rbac.Role(),
		rbac.RoleBinding(),
		rbac.ServiceAccount(),
		secret.New(),
		webhook.Issuer(),
		webhook.Certificate(),
		webhook.ValidatingWebhook(),
		webhook.MutatingWebhook(),
	).WithDefaultProcessor(processor.Default())
	for obj := range objects {
		appCtx.Add(obj)
	}
	return appCtx.CreateHelm(ctx.Done())
}

func setLogLevel(config config.Config) {
	logrus.SetLevel(logrus.ErrorLevel)
	if config.Verbose {
		logrus.SetLevel(logrus.InfoLevel)
	}
	if config.VeryVerbose {
		logrus.SetLevel(logrus.DebugLevel)
	}
}
