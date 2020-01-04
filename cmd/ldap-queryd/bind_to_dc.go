package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gopkg.in/ldap.v3"
)

//TODO Allow TLS bind
func bindToDC(directory directory, logger *logrus.Logger) (*ldap.Conn, error) {
	var ldapConn *ldap.Conn

	// Randomise the selection of hosts to allow for rudimentary load balancing of requests
	r := rand.New(rand.NewSource(time.Now().Unix()))

	for _, i := range r.Perm(len(directory.Hosts)) {
		ds := directory.Hosts[i]

		logger.WithFields(logrus.Fields{
			"ds":   ds,
			"port": directory.Port,
		}).Debug("attempting to connect to directory")

		var err error

		ldapConn, err = ldap.Dial("tcp", fmt.Sprintf("%s:%d", ds, directory.Port))
		if err != nil {
			logger.WithFields(logrus.Fields{
				"DC":       ds,
				"port":     directory.Port,
				"function": "bindToDC",
				"error":    err,
			}).Error("unable to dial LDAP directory server")

			continue
		}

		break
	}

	if ldapConn == nil {
		return nil, errors.New("unable to open connection to directory")
	}

	err := ldapConn.Bind(directory.BindDN, directory.BindPW)
	if err != nil {
		// Let's ensure we return a friendly error message if available
		if err, ok := err.(*ldap.Error); ok {
			return nil, errors.New(ldap.LDAPResultCodeMap[err.ResultCode])
		}

		return nil, errors.Wrap(err, "unable to bind to directory")
	}

	return ldapConn, nil
}
