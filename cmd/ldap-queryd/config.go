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
	Port               int
	Debug              bool
	AllowedSources     []string
	CorsAllowedOrigins []string
	CorsAllowedHeaders []string
}

type directory struct {
	Hosts  []string
	BindDN string
	BindPW string
	Port   int
}

func parseConfig(logger *logrus.Entry, allowedSources string, port int, debug bool, directoryHosts string, directoryBindDn string, directoryBindPwd string, directoryPort int, corsAllowedOrigins string, corsAllowedHeaders string) config {
	sources := strings.Replace(allowedSources, " ", "", -1)

	hosts := strings.Replace(directoryHosts, " ", "", -1)

	var allowedOrigins []string
	tmp := strings.Split(corsAllowedOrigins, ",")
	for _, v := range tmp {
		if strings.TrimSpace(v) == "" {
			continue
		}

		allowedOrigins = append(allowedOrigins, strings.TrimSpace(v))
	}

	var allowedHeaders []string
	tmp = strings.Split(corsAllowedHeaders, ",")
	for _, v := range tmp {
		allowedHeaders = append(allowedHeaders, strings.TrimSpace(v))
	}

	return config{
		Server: server{
			Port:               port,
			Debug:              debug,
			AllowedSources:     strings.Split(sources, ","),
			CorsAllowedOrigins: allowedOrigins,
			CorsAllowedHeaders: allowedHeaders,
		},
		Directory: directory{
			Hosts:  strings.Split(hosts, ","),
			BindDN: directoryBindDn,
			BindPW: directoryBindPwd,
			Port:   directoryPort,
		},
	}
}
