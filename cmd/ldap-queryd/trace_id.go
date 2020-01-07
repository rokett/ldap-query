package main

import (
	"context"
	"net/http"

	"github.com/gofrs/uuid"
	"github.com/justinas/alice"
	"github.com/sirupsen/logrus"
)

func traceID(logger *logrus.Entry) alice.Constructor {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			traceID, err := uuid.NewV4()
			if err != nil {
				logger.WithFields(logrus.Fields{
					"trace_id": traceID,
					"function": "search",
					"error":    err,
				}).Error("unable to generate trace ID")

				APIResponse := Response{
					Message: "unable to generate trace ID",
					Error:   err.Error(),
				}

				APIResponse.Send(http.StatusInternalServerError, w)

				return
			}

			ctx := context.WithValue(r.Context(), traceIDCtxKey, traceID.String())

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
