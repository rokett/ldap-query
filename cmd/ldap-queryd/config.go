package main

import (
	"strings"

	"github.com/sirupsen/logrus"
)

type config struct {
	Server    server
	Directory directory
}

type server struct {
	Port           int
	Debug          bool
	AllowedSources []string
}

type directory struct {
	Hosts  []string
	BindDN string
	BindPW string
	Port   int
}

func parseConfig(logger *logrus.Entry, allowedSources string, port int, debug bool, directoryHosts string, directoryBindDn string, directoryBindPwd string, directoryPort int) config {
	sources := strings.Replace(allowedSources, " ", "", -1)

	hosts := strings.Replace(directoryHosts, " ", "", -1)

	return config{
		Server: server{
			Port:           port,
			Debug:          debug,
			AllowedSources: strings.Split(sources, ","),
		},
		Directory: directory{
			Hosts:  strings.Split(hosts, ","),
			BindDN: directoryBindDn,
			BindPW: directoryBindPwd,
			Port:   directoryPort,
		},
	}
}
