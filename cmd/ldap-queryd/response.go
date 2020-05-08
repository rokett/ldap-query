package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// Response represents the API response content
type Response struct {
	Message string       `json:"message,omitempty"`
	Error   string       `json:"error,omitempty"`
	TraceID string       `json:"trace_id,omitempty"`
	Result  []ldapObject `json:"result,omitempty"`
}

// Send API response back to client
func (r Response) Send(httpStatus int, w http.ResponseWriter) {
	json, err := json.Marshal(r)
	if err != nil {
		fmt.Println("send error")
		fmt.Println(err)
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(httpStatus)
	w.Write(json)
}
