package main

import (
	"context"
	"flag"
	"io"
	"os"
	"path"

	"github.com/BurntSushi/toml"
	"github.com/besrabasant/k10ls/internal"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	klog "k8s.io/klog/v2"
)

func init() {
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
		ForceColors:   true,
	})

	// Silence verbose logs emitted by the Kubernetes libraries. By default
	// they use klog and utilruntime which print errors to stderr. These
	// lines suppress that output and instead log at debug level when
	// enabled.
	klog.InitFlags(nil)
	klog.LogToStderr(false)
	klog.SetOutput(io.Discard)
	utilruntime.ErrorHandlers = []utilruntime.ErrorHandler{
		func(_ context.Context, err error, msg string, _ ...interface{}) {
			if err != nil {
				logrus.Debugf("%s: %v", msg, err)
			}
		},
	}
}

func main() {
	// Set up CLI and config file handling with Viper
	configFile := flag.String("config", "config.toml", "Path to the config file")
	flag.Parse()

	viper.SetConfigFile(*configFile)
	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err != nil {
		logrus.Fatalf("Error reading config file: %v", err)
	}

	var config internal.Config
	if _, err := toml.DecodeFile(*configFile, &config); err != nil {
		logrus.Fatalf("Error parsing TOML config: %v", err)
	}

	if config.GlobalKubeConfig == "" {
		homedir, err := os.UserHomeDir()

		if err != nil {
			logrus.Fatal("error resolving user home directory.")
			os.Exit(1)
		}

		config.GlobalKubeConfig = path.Join(homedir, ".kube", "config")
	}

	// Iterate over each context
	for _, ctx := range config.Contexts {
		go internal.Portforward(&ctx, &config)
	}

	// Keep the process alive
	select {}
}
