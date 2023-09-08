package main

import (
	"os"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var (
	lg    = logrus.New()
	isK8s = os.Getenv("KUBERNETES_SERVICE_HOST") != ""
)

func setupConfig() *viper.Viper {
	cfg := viper.New()
	cfg.AddConfigPath(".")
	cfg.AddConfigPath("$HOME/kube-dnstap")
	cfg.AddConfigPath("/etc/")

	cfg.SetConfigName("kube-dnstap")

	cfg.SetDefault("name", "KUBE-DNSTAP")

	cfg.AutomaticEnv()
	cfg.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	cfg.SetDefault("listen.addr", "0.0.0.0:12345")
	cfg.SetDefault("metrics.addr", "0.0.0.0:8080")
	cfg.SetDefault("suffixes.ignore", []string{
		".svc.cluster.local.",
		".cluster.local.",
	})

	if isK8s {
		lg.SetFormatter(&logrus.JSONFormatter{})
	}

	if err := cfg.ReadInConfig(); err != nil {
		lg.WithError(err).Error("could not read initial config")
	}

	cfg.OnConfigChange(func(_ fsnotify.Event) {
		if err := cfg.ReadInConfig(); err != nil {
			lg.WithError(err).Warn("could not reload config")
		}
	})

	go cfg.WatchConfig()

	return cfg
}
