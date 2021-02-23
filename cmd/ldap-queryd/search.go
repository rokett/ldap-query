package main

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sirupsen/logrus"
	ldap "gopkg.in/ldap.v3"
)

type ldapObject struct {
	DistinguishedName string            `json:"distinguishedName,omitempty"`
	Attributes        map[string]string `json:"attributes,omitempty"`
}

var (
	requestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "ldapquery_request_duration_seconds",
			Help: "Time taken to query directory, partitioned by status code",
		},
		[]string{
			"status_code",
		},
	)

	queryError = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ldapquery_errors_total",
			Help: "Count of errors when querying directory, partitioned by operation, status code and client IP",
		},
		[]string{
			"operation",
			"status_code",
			"client",
		},
	)
)

func search(directory directory, logger *logrus.Entry) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The traceID is included in every log entry, and in HTTP responses, to allow for correlation of logs
		traceID := r.Context().Value(traceIDCtxKey).(string)

		APIResponse := Response{
			TraceID: traceID,
		}

		// The clientIP is included in every log entry and in some metrics for later analysis
		clientIP := r.Context().Value(clientIPCtxKey).(string)

		ldapConn, err := bindToDC(directory, logger)
		if err != nil {
			logger.WithFields(logrus.Fields{
				"trace_id":  traceID,
				"client_ip": clientIP,
				"function":  "search",
				"error":     err,
			}).Error("unable to bind to directory")

			APIResponse.Message = "unable to bind to directory"
			APIResponse.Error = err.Error()
			APIResponse.Send(http.StatusInternalServerError, w)

			return
		}
		defer ldapConn.Close()

		start := time.Now()

		body, err := io.ReadAll(r.Body)
		if err != nil {
			queryError.WithLabelValues("decode", strconv.Itoa(http.StatusInternalServerError), clientIP).Inc()

			logger.WithFields(logrus.Fields{
				"trace_id":  traceID,
				"client_ip": clientIP,
				"function":  "search",
				"error":     err,
			}).Error("unable to read HTTP request body")

			APIResponse.Message = "unable to read HTTP request body"
			APIResponse.Error = err.Error()
			APIResponse.Send(http.StatusInternalServerError, w)

			return
		}

		query := Query{}
		err = json.Unmarshal(body, &query)
		if err != nil {
			queryError.WithLabelValues("parse", strconv.Itoa(http.StatusInternalServerError), clientIP).Inc()

			logger.WithFields(logrus.Fields{
				"trace_id":  traceID,
				"client_ip": clientIP,
				"function":  "search",
				"error":     err,
				"query":     body,
			}).Error("unable to decode JSON query payload")

			APIResponse.Message = "unable to decode JSON query payload"
			APIResponse.Error = err.Error()
			APIResponse.Send(http.StatusInternalServerError, w)

			return
		}

		logger.WithFields(logrus.Fields{
			"trace_id":  traceID,
			"client_ip": clientIP,
			"function":  "search",
			"query":     body,
		}).Debug("Validate query")

		// We need to carry out some validation that the query passed by the user is actually valid.
		// We can't validate the filter, but we can validate that required fields are included and that
		// valid values have been passed for those fields which expect them.
		ve, err := query.Validate()
		if err != nil {
			json, err := json.Marshal(ve)
			if err != nil {
				queryError.WithLabelValues("validate", strconv.Itoa(http.StatusInternalServerError), clientIP).Inc()

				logger.WithFields(logrus.Fields{
					"trace_id":  traceID,
					"client_ip": clientIP,
					"function":  "search",
					"error":     err,
				}).Error("unable to encode errors from validation process")

				APIResponse.Message = "unable to encode errors from validation process"
				APIResponse.Error = err.Error()
				APIResponse.Send(http.StatusInternalServerError, w)

				return
			}

			logger.WithFields(logrus.Fields{
				"trace_id":          traceID,
				"client_ip":         clientIP,
				"function":          "search",
				"validation errors": json,
			}).Error("error(s) when validating incoming query")

			w.Header().Set("Content-Type", "application/json; charset=UTF-8")
			w.WriteHeader(http.StatusBadRequest)
			w.Write(json)

			queryError.WithLabelValues("validate", strconv.Itoa(http.StatusBadRequest), clientIP).Inc()

			return
		}

		// The ldap package defines the scopes as int, so we need to create a mapping between the string representation we're allowing
		// consumers of this service to send, and the ldap package constants.
		scopes := make(map[string]int)
		scopes["base"] = ldap.ScopeBaseObject
		scopes["one"] = ldap.ScopeSingleLevel
		scopes["sub"] = ldap.ScopeWholeSubtree

		searchRequest := ldap.NewSearchRequest(
			query.Base,
			scopes[query.Scope],
			ldap.NeverDerefAliases,
			0,
			0,
			false,
			query.Filter,
			query.Attributes,
			nil,
		)

		var pageSize uint32
		pageSize = 10000
		res, err := ldapConn.SearchWithPaging(searchRequest, pageSize)
		if err != nil {
			err2 := err
			// Let's ensure we return a friendly error message if available
			if err, ok := err.(*ldap.Error); ok {
				err2 = errors.New(ldap.LDAPResultCodeMap[err.ResultCode])
			}

			queryError.WithLabelValues("search", strconv.Itoa(http.StatusInternalServerError), clientIP).Inc()

			logger.WithFields(logrus.Fields{
				"trace_id":   traceID,
				"client_ip":  clientIP,
				"function":   "search",
				"error":      err2,
				"filter":     query.Filter,
				"attributes": query.Attributes,
				"scope":      query.Scope,
				"base":       query.Base,
			}).Error("unable to search LDAP")

			APIResponse.Message = "unable to search LDAP"
			APIResponse.Error = err2.Error()
			APIResponse.Send(http.StatusInternalServerError, w)

			return
		}

		var objects []ldapObject

		// The response object contains a slice of returned entries.
		// We need to loop over them and pull out any attributes requested by the consumer.
		for _, entry := range res.Entries {
			var object ldapObject

			object.Attributes = make(map[string]string)

			for _, a := range query.Attributes {
				if strings.ToLower(a) == "distinguishedname" {
					object.DistinguishedName = entry.DN
					continue
				}

				object.Attributes[a] = entry.GetAttributeValue(a)
			}

			objects = append(objects, object)
		}

		duration := time.Since(start)
		requestDuration.WithLabelValues(strconv.Itoa(http.StatusOK)).Observe(duration.Seconds())

		if len(objects) == 0 {
			APIResponse.Send(http.StatusNotFound, w)
			return
		}

		APIResponse.Result = objects
		APIResponse.Send(http.StatusOK, w)
	})
}
