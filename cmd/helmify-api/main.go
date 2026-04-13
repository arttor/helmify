package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/arttor/helmify/pkg/app"
	"github.com/arttor/helmify/pkg/config"
	"github.com/arttor/helmify/pkg/helm"
	"github.com/arttor/helmify/pkg/translator/k8smanifest"
	"github.com/sirupsen/logrus"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/generate", handleGenerate)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		logrus.Infof("Starting Helmify API on %s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.Fatalf("listen: %s\n", err)
		}
	}()

	<-done
	logrus.Info("Server Stopping...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logrus.Fatalf("Server Shutdown Failed:%+v", err)
	}
	logrus.Info("Server Exited Properly")
}

func handleGenerate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	conf := config.Config{
		ChartName: r.Header.Get("X-Chart-Name"),
	}
	if conf.ChartName == "" {
		conf.ChartName = "chart"
	}

	// Simple header parsing for booleans
	conf.Crd, _ = strconv.ParseBool(r.Header.Get("X-Crd"))
	conf.CertManagerAsSubchart, _ = strconv.ParseBool(r.Header.Get("X-Cert-Manager-Subchart"))
	conf.CertManagerInstallCRD, _ = strconv.ParseBool(r.Header.Get("X-Cert-Manager-Install-Crd"))
	conf.AddWebhookOption, _ = strconv.ParseBool(r.Header.Get("X-Add-Webhook-Option"))
	conf.OptionalCRDs, _ = strconv.ParseBool(r.Header.Get("X-Optional-Crds"))
	conf.CertManagerVersion = r.Header.Get("X-Cert-Manager-Version")
	if conf.CertManagerVersion == "" {
		conf.CertManagerVersion = "v1.11.0"
	}

	logrus.Infof("Generating chart: %s", conf.ChartName)

	memOut := helm.NewMemoryOutput()
	engine := app.NewEngine(conf, memOut)
	trans := k8smanifest.New(conf, r.Body)

	if err := engine.Run(r.Context(), trans); err != nil {
		logrus.WithError(err).Error("Engine failed")
		http.Error(w, fmt.Sprintf("Failed to generate chart: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/x-tar")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.tar.gz"`, conf.ChartName))

	if err := memOut.ToTarGz(conf.ChartName, w); err != nil {
		logrus.WithError(err).Error("TarGz failed")
		// Note: we might have already sent some data, so we can't http.Error here reliably
	}
}
