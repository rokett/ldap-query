package main

import (
	"context"
	"net"
	"net/http"
	"strings"
)

func getClientIP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Retrieve the client IP from the remote address of the request
		clientIP, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			APIResponse := Response{
				Message: "unable to retrieve 'RemoteAddr' HTTP header",
				Error:   err.Error(),
			}

			APIResponse.Send(http.StatusInternalServerError, w)

			return
		}

		// If the request has passed through a proxy, the RemoteAddr may actually end up being the IP of the proxy.
		// Depending on how the proxy is configured, the X-Forwarded-For header may have been inserted to show the REAL client IP.
		// If it's there, we'll grab it and use it.
		// X-Forwarded-For could be a comma separated list, so we'll split it and then select the first entry which is the real client IP.
		// See https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/X-Forwarded-For for more info.
		if r.Header.Get("X-Forwarded-For") != "" {
			xForwardedFor := strings.Split(r.Header.Get("X-Forwarded-For"), ",")
			clientIP = strings.TrimSpace(xForwardedFor[0])
		}

		ctx := context.WithValue(r.Context(), clientIPCtxKey, clientIP)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
