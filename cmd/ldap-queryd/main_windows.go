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
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/eventlog"
)

type adQueryContextKeyType string

const clientIPCtxKey adQueryContextKeyType = "client_ip"
const traceIDCtxKey adQueryContextKeyType = "trace_id"

var (
	app     = "LDAP-Query"
	version string
	build   string
)

func main() {
	var (
		serverPort = flag.Int("server-port", 9999, "API Gateway server port")
		versionFlg = flag.Bool("version", false, "Display application version")
		debug      = flag.Bool("debug", false, "Enable debugging?")
	)

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

	// If the application is NOT running interactively, then it is running as a service and we want to send logs to the Event Log.
	interactive, err := svc.IsAnInteractiveSession()
	if err != nil {
		logger.WithFields(logrus.Fields{
			"function": "main",
			"error":    err,
		}).Fatal("unable to determine if process is running interactively")
	}
	if interactive == false {
		var el *eventlog.Log

		el, err := eventlog.Open(app)
		if err != nil {
			logger.WithFields(logrus.Fields{
				"function": "main",
				"error":    err,
			}).Error("unable to open Event Log")

			err := eventlog.InstallAsEventCreate(app, eventlog.Error|eventlog.Warning|eventlog.Info)
			if err != nil {
				logger.WithFields(logrus.Fields{
					"function": "main",
					"error":    err,
				}).Fatal("unable to create event source")
			}

			el, err = eventlog.Open(app)
			if err != nil {
				logger.WithFields(logrus.Fields{
					"function": "main",
					"error":    err,
				}).Fatal("unable to open event log after creating event source")
			}
		}
		defer el.Close()

		logger.Hooks.Add(eventloghook.NewHook(el))
	}

	config := loadConfig(logger)

	// Need to ensure that we can bind to the directory before we bother listening for any requests.
	ldapConn, err := bindToDC(config.Directory, logger)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"function": "main",
			"error":    err,
		}).Fatal("unable to bind to directory")
	}
	ldapConn.Close()

	listeningPort := ":" + strconv.Itoa(*serverPort)
	server, err := net.Listen("tcp", listeningPort)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"port":  listeningPort,
			"error": err,
		}).Fatal("unable to listen")
	}
	defer server.Close()

	middlewareChain := alice.New(
		checkMethodIsPOST, // Ensure method is allowed
		getClientIP,       // Store original client IP address in context
		checkRequestSource(config.AllowedSources, logger), // Ensure source IP is allowed to query
		traceID(logger), // Generate Trace ID and store in context
	)

	// TODO Use custom servemux

	http.Handle("/", middlewareChain.ThenFunc(search(config.Directory, logger)))
	http.Handle("/metrics", promhttp.Handler())

	logger.WithField("port", listeningPort).Debug("API server listening")

	err = http.Serve(server, nil)
	if err != nil {
		logger.WithField("error", err).Fatal("unable to start server")
	}
}
