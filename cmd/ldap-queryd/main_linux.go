package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"

	"github.com/justinas/alice"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

type adQueryContextKeyType string

const clientIPCtxKey adQueryContextKeyType = "client_ip"
const traceIDCtxKey adQueryContextKeyType = "trace_id"

var (
	app           = "LDAP-Query"
	version       string
	build         string
	directoryPort = 389

	versionFlg          = flag.Bool("version", false, "Display application version")
	portFlg             = flag.Int("port", 9999, "Port to listen for requests on")
	debugFlg            = flag.Bool("debug", false, "Enable debug logging")
	allowedSourcesFlg   = flag.String("allowed_sources", "", "IPs for sources that need to be able to make queries")
	directoryHostsFlg   = flag.String("directory_hosts", "", "LDAP hosts to query")
	directoryBindDnFlg  = flag.String("directory_bind_dn", "", "DN of account used to bind to the directory")
	directoryBindPwdFlg = flag.String("directory_bind_pw", "", "Password for account used to bind to the directory")
	helpFlg             = flag.Bool("help", false, "Display application help")
)

func main() {
	flag.Parse()

	if *versionFlg {
		fmt.Printf("%s v%s build %s\n", app, version, build)
		os.Exit(0)
	}

	if *helpFlg {
		flag.PrintDefaults()
		os.Exit(0)
	}

	logrusLogger := logrus.New()

	logrusLogger.Out = os.Stdout
	logrusLogger.Formatter = &logrus.JSONFormatter{}

	logger := logrusLogger.WithFields(logrus.Fields{
		"version": version,
		"build":   build,
	})

	missingFlags := false

	if *allowedSourcesFlg == "" {
		missingFlags = true
		logger.WithFields(logrus.Fields{
			"function": "run",
		}).Error("The allowed_sources flag is required")
	}

	if *directoryHostsFlg == "" {
		missingFlags = true
		logger.WithFields(logrus.Fields{
			"function": "run",
		}).Error("The directory_hosts flag is required")
	}

	if *directoryBindDnFlg == "" {
		missingFlags = true
		logger.WithFields(logrus.Fields{
			"function": "run",
		}).Error("The directory_bind_dn flag is required")
	}

	if *directoryBindPwdFlg == "" {
		missingFlags = true
		logger.WithFields(logrus.Fields{
			"function": "run",
		}).Error("The directory_bind_pw flag is required")
	}

	if missingFlags {
		flag.PrintDefaults()
		os.Exit(1)
	}

	config := parseConfig(logger, *allowedSourcesFlg, *portFlg, *debugFlg, *directoryHostsFlg, *directoryBindDnFlg, *directoryBindPwdFlg, directoryPort)

	if config.Server.Debug {
		logger.Level = logrus.DebugLevel
	} else {
		logger.Level = logrus.InfoLevel
	}

	// Need to ensure that we can bind to the directory before we bother listening for any requests.
	ldapConn, err := bindToDC(config.Directory, logger)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"function": "main",
			"error":    err,
		}).Fatal("unable to bind to directory")
	}
	ldapConn.Close()

	listeningPort := fmt.Sprintf(":%d", config.Server.Port)
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
		checkRequestSource(config.Server.AllowedSources, logger), // Ensure source IP is allowed to query
		traceID(logger), // Generate Trace ID and store in context
	)

	// Using a locally scoped ServerMux to ensure that the only routes that can be registered are our own
	mux := http.NewServeMux()

	mux.Handle("/", middlewareChain.ThenFunc(search(config.Directory, logger)))
	mux.Handle("/metrics", promhttp.Handler())

	logger.WithField("port", listeningPort).Debug("API server listening")

	err = http.Serve(server, mux)
	if err != nil {
		logger.WithField("error", err).Fatal("unable to start server")
	}
}
