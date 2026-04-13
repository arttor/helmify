package app

import (
	"context"

	"github.com/arttor/helmify/pkg/processor/job"
	"github.com/arttor/helmify/pkg/processor/poddisruptionbudget"
	"github.com/arttor/helmify/pkg/processor/statefulset"

	"github.com/sirupsen/logrus"

	"github.com/arttor/helmify/pkg/config"
	"github.com/arttor/helmify/pkg/helmify"
	"github.com/arttor/helmify/pkg/processor"
	"github.com/arttor/helmify/pkg/processor/configmap"
	"github.com/arttor/helmify/pkg/processor/crd"
	"github.com/arttor/helmify/pkg/processor/daemonset"
	"github.com/arttor/helmify/pkg/processor/deployment"
	"github.com/arttor/helmify/pkg/processor/rbac"
	"github.com/arttor/helmify/pkg/processor/secret"
	"github.com/arttor/helmify/pkg/processor/service"
	"github.com/arttor/helmify/pkg/processor/storage"
	"github.com/arttor/helmify/pkg/processor/webhook"
	"github.com/arttor/helmify/pkg/translator"
	"github.com/arttor/helmify/pkg/processor/route"
)

// Engine is the core helmify processing engine, decoupled from inputs like stdin or files.
type Engine struct {
	config config.Config
	output helmify.Output
}

// NewEngine creates a new processing Engine.
func NewEngine(cfg config.Config, output helmify.Output) *Engine {
	return &Engine{
		config: cfg,
		output: output,
	}
}

// Run executes the translation pipeline and creates a Helm chart.
func (e *Engine) Run(ctx context.Context, trans translator.Translator) error {
	err := e.config.Validate()
	if err != nil {
		return err
	}
	setLogLevel(e.config)

	appCtx := New(e.config, e.output)
	appCtx = appCtx.WithProcessors(
		configmap.New(),
		crd.New(),
		daemonset.New(),
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
		job.NewCron(),
		job.NewJob(),
		poddisruptionbudget.New(),
		route.New(),
	).WithDefaultProcessor(processor.Default())

	payloads, err := trans.Translate(ctx)
	if err != nil {
		return err
	}

	for payload := range payloads {
		cleanKomposeMetadata(payload.Object)
		appCtx.Add(payload.Object, payload.Filename)
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
