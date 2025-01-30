package main

import (
	"flag"
	"os"
	"path"

	"github.com/BurntSushi/toml"
	"github.com/besrabasant/k10ls/internal"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func init()  {
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
		ForceColors: true,
	})
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

