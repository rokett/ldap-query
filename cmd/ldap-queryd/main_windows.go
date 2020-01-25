package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"

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
	app           = "LDAP-Query"
	version       string
	build         string
	serviceDesc   = "REST API gateway for running queries against LDAP directory"
	directoryPort = 389

	versionFlg          = flag.Bool("version", false, "Display application version")
	winServiceCommand   = flag.String("service", "", "Manage Windows services: install, uninstall, start, stop")
	portFlg             = flag.Int("port", 9999, "Port to listen for requests on")
	debugFlg            = flag.Bool("debug", false, "Enable debug logging")
	allowedSourcesFlg   = flag.String("allowed_sources", "", "IPs for sources that need to be able to make queries")
	directoryHostsFlg   = flag.String("directory_hosts", "", "LDAP hosts to query")
	directoryBindDnFlg  = flag.String("directory_bind_dn", "", "DN of account used to bind to the directory")
	directoryBindPwdFlg = flag.String("directory_bind_pw", "", "Password for account used to bind to the directory")
	helpFlg             = flag.Bool("help", false, "Display application help")
)

type program struct {
	logger *logrus.Entry
}

//TODO Tracing

func main() {
	logrusLogger := logrus.New()

	logrusLogger.Out = os.Stdout
	logrusLogger.Formatter = &logrus.JSONFormatter{}

	logger := logrusLogger.WithFields(logrus.Fields{
		"version": version,
		"build":   build,
	})

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

		p.logger.Logger.Hooks.Add(eventloghook.NewHook(el))
	}

	flag.Parse()

	if *versionFlg {
		fmt.Printf("%s v%s build %s\n", app, version, build)
		os.Exit(0)
	}

	if *helpFlg {
		flag.PrintDefaults()
		os.Exit(0)
	}

	missingFlags := false

	if *allowedSourcesFlg == "" {
		missingFlags = true
		p.logger.WithFields(logrus.Fields{
			"function": "run",
		}).Error("The allowed_sources flag is required")
	}

	if *directoryHostsFlg == "" {
		missingFlags = true
		p.logger.WithFields(logrus.Fields{
			"function": "run",
		}).Error("The directory_hosts flag is required")
	}

	if *directoryBindDnFlg == "" {
		missingFlags = true
		p.logger.WithFields(logrus.Fields{
			"function": "run",
		}).Error("The directory_bind_dn flag is required")
	}

	if *directoryBindPwdFlg == "" {
		missingFlags = true
		p.logger.WithFields(logrus.Fields{
			"function": "run",
		}).Error("The directory_bind_pw flag is required")
	}

	if missingFlags {
		flag.PrintDefaults()
		os.Exit(1)
	}

	config := parseConfig(p.logger, *allowedSourcesFlg, *portFlg, *debugFlg, *directoryHostsFlg, *directoryBindDnFlg, *directoryBindPwdFlg, directoryPort)

	if config.Server.Debug {
		p.logger.Logger.Level = logrus.DebugLevel
	} else {
		p.logger.Logger.Level = logrus.InfoLevel
	}

	// Need to ensure that we can bind to the directory before we bother listening for any requests.
	ldapConn, err := bindToDC(config.Directory, p.logger)
	if err != nil {
		p.logger.WithFields(logrus.Fields{
			"function": "run",
			"error":    err,
		}).Fatal("unable to bind to directory")
	}
	ldapConn.Close()

	listeningPort := fmt.Sprintf(":%d", config.Server.Port)
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
		checkRequestSource(config.Server.AllowedSources, p.logger), // Ensure source IP is allowed to query
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
