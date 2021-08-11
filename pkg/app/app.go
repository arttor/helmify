package app

import (
	"context"
	"github.com/arttor/helmify/pkg/config"
	"github.com/arttor/helmify/pkg/decoder"
	"github.com/arttor/helmify/pkg/helm"
	"github.com/arttor/helmify/pkg/helmify"
	"github.com/arttor/helmify/pkg/processor/configmap"
	"github.com/arttor/helmify/pkg/processor/crd"
	"github.com/arttor/helmify/pkg/processor/deployment"
	"github.com/arttor/helmify/pkg/processor/rbac"
	"github.com/arttor/helmify/pkg/processor/secret"
	"github.com/arttor/helmify/pkg/processor/service"
	"github.com/arttor/helmify/pkg/processor/webhook"
	"github.com/sirupsen/logrus"
	"io"
	"os"
	"os/signal"
	"syscall"
)

func Start(input io.Reader, config config.Config) error {
	if !config.Verbose {
		logrus.SetLevel(logrus.ErrorLevel)
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
	objects := decoder.Decode(ctx.Done(), input)
	helmifyContext := &helmify.Context{}
	helmifyContext = helmifyContext.WithConfig(config).WithProcessors(configmap.New(),
		crd.New(),
		deployment.New(),
		service.New(),
		rbac.ClusterRole(),
		rbac.ClusterRoleBinding(),
		rbac.Role(),
		rbac.RoleBinding(),
		rbac.ServiceAccount(),
		webhook.Issuer(),
		webhook.Certificate(),
		secret.Secret(),
		webhook.Webhook()).WithOutput(helm.NewOutput())

	for obj := range objects {
		helmifyContext.Add(obj)
	}
	return helmifyContext.CreateHelm(ctx.Done())
}
