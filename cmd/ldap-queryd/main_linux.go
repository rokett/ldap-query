package main

import (
	"flag"
	"net"
	"net/http"
	"os"
	"strconv"

	"github.com/justinas/alice"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

type adQueryContextKeyType string

const clientIPCtxKey adQueryContextKeyType = "client_ip"
const traceIDCtxKey adQueryContextKeyType = "trace_id"

func main() {
	var (
		serverPort = flag.Int("server-port", 9999, "API Gateway server port")
		debug      = flag.Bool("debug", false, "Enable debugging?")
	)

	flag.Parse()

	logger := logrus.New()

	logger.Out = os.Stdout
	logger.Formatter = &logrus.JSONFormatter{}

	if *debug {
		logger.Level = logrus.DebugLevel
	} else {
		logger.Level = logrus.InfoLevel
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
