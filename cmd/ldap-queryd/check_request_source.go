package main

import (
	"fmt"
	"net/http"

	"github.com/justinas/alice"
	"github.com/sirupsen/logrus"
)

func checkRequestSource(allowedSources []string, logger *logrus.Entry) alice.Constructor {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			allowed := false

			for _, ip := range allowedSources {
				if ip == r.Context().Value(clientIPCtxKey) {
					allowed = true
					break
				}
			}

			if !allowed {
				msg := fmt.Sprintf("%s is not allowed to query; check the config", r.Context().Value(clientIPCtxKey))
				APIResponse := Response{
					Message: msg,
				}

				APIResponse.Send(http.StatusUnauthorized, w)

				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
