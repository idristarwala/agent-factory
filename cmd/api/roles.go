package main

import (
	"crypto/subtle"
	"net/http"
	"os"
	"strings"
)

type apiRole int

const (
	viewer apiRole = iota
	operator
	admin
)

type credentials [3]string

func apiCredentials() credentials {
	return credentials{viewer: os.Getenv("FACTORY_VIEWER_TOKEN"), operator: os.Getenv("FACTORY_OPERATOR_TOKEN"), admin: os.Getenv("API_TOKEN")}
}

func (c credentials) enabled() bool { return c != (credentials{}) }

func (c credentials) valid() bool {
	for i, token := range c {
		if token == "" {
			continue
		}
		for _, previous := range c[:i] {
			if previous == token {
				return false
			}
		}
	}
	return true
}

func requireRole(w http.ResponseWriter, r *http.Request, required apiRole) bool {
	c := apiCredentials()
	if !c.enabled() {
		return true
	}
	if !c.valid() {
		http.Error(w, "invalid API role configuration", http.StatusServiceUnavailable)
		return false
	}
	token, bearer := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	if bearer && token != "" {
		for role, configured := range c {
			if subtle.ConstantTimeCompare([]byte(token), []byte(configured)) == 1 {
				if apiRole(role) >= required {
					return true
				}
				http.Error(w, "forbidden", http.StatusForbidden)
				return false
			}
		}
	}
	w.Header().Set("WWW-Authenticate", "Bearer")
	http.Error(w, "unauthorized", http.StatusUnauthorized)
	return false
}
