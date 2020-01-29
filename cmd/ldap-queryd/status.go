package main

import (
	"net/http"
)

func status() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		APIResponse := Response{
			Message: "ok",
		}

		APIResponse.Send(http.StatusOK, w)
	})
}
