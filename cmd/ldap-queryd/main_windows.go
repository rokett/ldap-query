package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"

	"github.com/Freman/eventloghook"
	"github.com/justinas/alice"
	"github.com/kardianos/service"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/windows/svc/eventlog"
)

type adQueryContextKeyType string

const clientIPCtxKey adQueryContextKeyType = "client_ip"
const traceIDCtxKey adQueryContextKeyType = "trace_id"

var (
	app         = "LDAP-Query"
	version     string
	build       string
	serviceDesc = "REST API gateway for running queries against LDAP directory"

	serverPort        = flag.Int("server-port", 9999, "API Gateway server port")
	versionFlg        = flag.Bool("version", false, "Display application version")
	debug             = flag.Bool("debug", false, "Enable debugging?")
	winServiceCommand = flag.String("service", "", "Manage Windows services: install, uninstall, start, stop")
)

type program struct {
	logger *logrus.Logger
}

//TODO Tracing
//TODO Include app version in logs

func main() {
	flag.Parse()

	if *versionFlg {
		fmt.Printf("%s v%s build %s\n", app, version, build)
		os.Exit(0)
	}

	logger := logrus.New()

	logger.Out = os.Stdout
	logger.Formatter = &logrus.JSONFormatter{}

	if *debug {
		logger.Level = logrus.DebugLevel
	} else {
		logger.Level = logrus.InfoLevel
	}

	svcConfig := &service.Config{
		Name:        app,
		DisplayName: app,
		Description: serviceDesc,
	}

	prg := &program{
		logger: logger,
	}

	svc, err := service.New(prg, svcConfig)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"function": "main",
			"error":    err,
		}).Fatal("error creating new service")
	}

	errs := make(chan error, 5)

	go func() {
		for {
			err := <-errs
			if err != nil {
				logger.WithFields(logrus.Fields{
					"function": "main",
					"error":    err,
				}).Fatal("unknown error")
			}
		}
	}()

	if *winServiceCommand != "" {
		err := service.Control(svc, *winServiceCommand)
		if err != nil {
			logger.WithFields(logrus.Fields{
				"function": "main",
				"error":    err,
				"control":  *winServiceCommand,
			}).Error("error controlling service")
		}

		os.Exit(0)
	} else {
		err = svc.Run()
		if err != nil {
			logger.WithFields(logrus.Fields{
				"function": "main",
				"error":    err,
			}).Error("error running application interactively")
		}
	}
}

func (p *program) Start(s service.Service) error {
	// Start should not block. Do the actual work async.
	go p.run(s)
	return nil
}

func (p *program) run(svc service.Service) {
	// If the application is NOT running interactively, then it is running as a service and we want to send logs to the Event Log.
	if service.Interactive() == false {
		var el *eventlog.Log

		el, err := eventlog.Open(app)
		if err != nil {
			p.logger.WithFields(logrus.Fields{
				"function": "run",
				"error":    err,
			}).Error("unable to open Event Log")

			err := eventlog.InstallAsEventCreate(app, eventlog.Error|eventlog.Warning|eventlog.Info)
			if err != nil {
				p.logger.WithFields(logrus.Fields{
					"function": "run",
					"error":    err,
				}).Fatal("unable to create event source")
			}

			el, err = eventlog.Open(app)
			if err != nil {
				p.logger.WithFields(logrus.Fields{
					"function": "run",
					"error":    err,
				}).Fatal("unable to open event log after creating event source")
			}
		}
		defer el.Close()

		p.logger.Hooks.Add(eventloghook.NewHook(el))
	}

	config := loadConfig(p.logger)

	// Need to ensure that we can bind to the directory before we bother listening for any requests.
	ldapConn, err := bindToDC(config.Directory, p.logger)
	if err != nil {
		p.logger.WithFields(logrus.Fields{
			"function": "run",
			"error":    err,
		}).Fatal("unable to bind to directory")
	}
	ldapConn.Close()

	listeningPort := ":" + strconv.Itoa(*serverPort)
	server, err := net.Listen("tcp", listeningPort)
	if err != nil {
		p.logger.WithFields(logrus.Fields{
			"port":  listeningPort,
			"error": err,
		}).Fatal("unable to listen")
	}
	defer server.Close()

	middlewareChain := alice.New(
		checkMethodIsPOST, // Ensure method is allowed
		getClientIP,       // Store original client IP address in context
		checkRequestSource(config.AllowedSources, p.logger), // Ensure source IP is allowed to query
		traceID(p.logger), // Generate Trace ID and store in context
	)

	// Using a locally scoped ServerMux to ensure that the only routes that can be registered are our own
	mux := http.NewServeMux()

	mux.Handle("/", middlewareChain.ThenFunc(search(config.Directory, p.logger)))
	mux.Handle("/metrics", promhttp.Handler())

	p.logger.WithFields(logrus.Fields{
		"function": "run",
		"port":     listeningPort,
	}).Debug("API server listening")

	err = http.Serve(server, mux)
	if err != nil {
		p.logger.WithFields(logrus.Fields{
			"function": "run",
			"error":    err,
		}).Fatal("unable to start server")
	}
}

func (p *program) Stop(s service.Service) error {
	p.logger.Info("Stopping")
	// Stop should not block. Return with a few seconds.
	return nil
}
