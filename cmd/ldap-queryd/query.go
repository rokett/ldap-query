package main

import (
	"encoding/json"
	"errors"
	"regexp"
)

// Query contains the possible parameters that can be passed in the request body when carrying out a search against an LDAP directory
// Filter = needs to be a valid LDAP filter
// Base = defines the base OU of the search
// Scope = one of base, one, or sub to define what is searched
// Attributes = array of strings with the attributes to return from the search
type Query struct {
	// REQUIRED parameter(s)
	Filter     string   `json:"filter"`
	Base       string   `json:"base"`
	Attributes []string `json:"attributes"`

	// OPTIONAL parameter(s)
	Scope string `json:"scope"`
}

// ValidationError contains the parameter with the error and a friendly error message
type ValidationError struct {
	Parameter string `json:"parameter"`
	Error     string `json:"error"`
}

// Validate ensures that the query passed is valid
func (q *Query) Validate() ([]ValidationError, error) {
	var ve []ValidationError

	regexScope := `(?i)^(base|one|sub)$`
	regexBase := `(?i)^(?:ou=[^,]*,?)*(?:dc=[^,]*,?)*$`

	// REQUIRED parameter validation
	if q.Filter == "" {
		ve = append(ve, ValidationError{
			Parameter: "filter",
			Error:     "REQUIRED field",
		})
	}

	if q.Base == "" {
		ve = append(ve, ValidationError{
			Parameter: "base",
			Error:     "REQUIRED field",
		})
	}

	if q.Base != "" && !regexp.MustCompile(regexBase).MatchString(q.Base) {
		ve = append(ve, ValidationError{
			Parameter: "base",
			Error:     "base does not appear to be a valid LDAP path",
		})
	}

	if len(q.Attributes) == 0 {
		ve = append(ve, ValidationError{
			Parameter: "attributes",
			Error:     "attributes to be returned MUST be defined",
		})
	}

	// OPTIONAL parameter validation
	if !regexp.MustCompile(regexScope).MatchString(q.Scope) {
		ve = append(ve, ValidationError{
			Parameter: "scope",
			Error:     "If specified, scope MUST be one of 'base', 'one', or 'sub'",
		})
	}

	if len(ve) > 0 {
		return ve, errors.New("validation failed")
	}

	return ve, nil
}

// UnmarshalJSON implements a custom unmarshaller for the query JSON payload to set default values if parameters have not been included.
func (q *Query) UnmarshalJSON(data []byte) error {
	// Set default values before unmarshaling
	q.Scope = "base"

	// Creating an Alias type prevents an endless loop
	type Alias Query
	tmp := (*Alias)(q)

	return json.Unmarshal(data, tmp)
}
