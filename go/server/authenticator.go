// Copyright (C) 2019 Cranky Kernel
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package server

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"gitlab.com/crankykernel/maker/go/auth"
	"gitlab.com/crankykernel/maker/go/config"
	"gitlab.com/crankykernel/maker/go/log"
	mathrand "math/rand"
	"net/http"
	"strings"
	"time"
)

func init() {
	mathrand.Seed(time.Now().UnixNano())
}

type Authenticator struct {
	username string
	password string
	sessions map[string]bool
}

func NewAuthenticator(configFilename string) *Authenticator {
	m := Authenticator{
		sessions: map[string]bool{},
	}

	m.username = config.GetString("username")
	if m.username == "" {
		m.username = "maker"
		config.Set("username", m.username)
	}

	showPassword := false
	password := config.GetString("password")
	if password != "" {
		m.password = password
	} else {
		password, m.password = m.generatePassword(configFilename)
		showPassword = true
	}

	if showPassword {
		fmt.Printf(`
A username and password have been generated for you. Please take note of them.
This is the one and only time the password will be available.

Username: %s
Password: %s

`, m.username, password)
	}

	return &m
}

func (m *Authenticator) hasSession(sessionId string) bool {
	val, ok := m.sessions[sessionId]
	if val && ok {
		return true
	}
	return false
}

func (m *Authenticator) generatePassword(configFilename string) (string, string) {
	password := m.getRandom(32)
	encoded, err := auth.EncodePassword(password)
	if err != nil {
		log.Fatal("Failed to encode generated password: %v", err)
	}
	config.Set("password", encoded)
	config.WriteConfig(configFilename)
	return password, encoded
}

func (m *Authenticator) getRandom(size int) string {
	alphanumerics := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890")
	b := make([]rune, size)
	for i := range b {
		b[i] = alphanumerics[mathrand.Intn(len(alphanumerics))]
	}
	return string(b)
}

func (m *Authenticator) requiresAuth(path string) bool {
	if strings.HasPrefix(path, "/api/login") {
		return false
	}
	if strings.HasPrefix(path, "/api") {
		return true
	}
	if strings.HasPrefix(path, "/ws") {
		return true
	}
	if strings.HasPrefix(path, "/proxy") {
		return true
	}
	return false
}

func (m *Authenticator) generateSessionId() (string, error) {
	bytes := make([]byte, 128)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func (m *Authenticator) Login(username string, password string) (string, error) {
	if username != m.username {
		return "", fmt.Errorf("bad username")
	}
	ok, err := auth.CheckPassword(password, m.password)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", fmt.Errorf("bad password")
	}
	sessionId, err := m.generateSessionId()
	if err != nil {
		return "", err
	}
	m.sessions[sessionId] = true
	return sessionId, nil
}

// Middleware function, which will be called for each request
func (m *Authenticator) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !m.requiresAuth(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		sessionId := r.FormValue("sessionId")
		if sessionId != "" {
			if m.hasSession(sessionId) {
				next.ServeHTTP(w, r)
				return
			}
		}

		sessionId = r.Header.Get("X-Session-ID")
		if sessionId != "" {
			if m.hasSession(sessionId) {
				next.ServeHTTP(w, r)
				return
			}
		}

		w.WriteHeader(http.StatusUnauthorized)
	})
}
