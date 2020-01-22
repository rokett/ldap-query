package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	packr "github.com/gobuffalo/packr/v2"
	"github.com/sirupsen/logrus"
)

type config struct {
	Server         server    `toml:"server"`
	Directory      directory `toml:"directory"`
}

type server struct {
	Port           int      `toml:"port"`
	Debug          bool     `toml:"debug"`
	AllowedSources []string `toml:"allowed_sources"`
}

type directory struct {
	Hosts         []string `toml:"hosts"`
	BindDN        string   `toml:"bind_dn"`
	BindPW        string   `toml:"bind_password"`
	Port          int      `toml:"port"`
	UseSSL        bool     `toml:"use_ssl"`
	StartTLS      bool     `toml:"start_tls"`
	SSLSkipVerify bool     `toml:"ssl_skip_verify"`
}

func loadConfig(logger *logrus.Entry) config {
	var (
		cfgFile = "config.toml"
		config  config
	)

	exe, err := os.Executable()
	if err != nil {
		logger.WithFields(logrus.Fields{
			"function": "loadConfig",
			"error":    err,
		}).Error("unable to get executable path")

		os.Exit(1)
	}
	path := filepath.Dir(exe)
	cfgFile = fmt.Sprintf("%s/%s", path, cfgFile)

	// Load/create the config file
	if _, err := os.Stat(cfgFile); os.IsNotExist(err) {
		logger.WithFields(logrus.Fields{
			"function": "loadConfig",
			"error":    err,
		}).Error("the configuration file does not exist; creating template config file")

		box := packr.New("templatesBox", "../../templates")

		tmpl, err := box.Find("config.example.toml")
		if err != nil {
			logger.WithFields(logrus.Fields{
				"function": "loadConfig",
				"error":    err,
			}).Error("unable to load ../../templates/config.example.toml")

			os.Exit(1)
		}

		f, err := os.Create(cfgFile)
		if err != nil {
			logger.WithFields(logrus.Fields{
				"function": "loadConfig",
				"error":    err,
			}).Errorf("unable to create config file; %s", cfgFile)

			os.Exit(1)
		}
		defer f.Close()

		_, err = f.Write(tmpl)
		if err != nil {
			logger.WithFields(logrus.Fields{
				"function": "loadConfig",
				"error":    err,
			}).Error("unable to write config template to file")

			os.Exit(1)
		}

		f.Sync()

		fmt.Println("Config file created.  Check the configuration before running this application again")

		os.Exit(0)
	}

	_, err = toml.DecodeFile(cfgFile, &config)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"function": "loadConfig",
			"error":    err,
		}).Error("unable to decode config file")

		os.Exit(1)
	}

	return config
}
