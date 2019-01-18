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
	"fmt"
	"gitlab.com/crankykernel/maker/go/auth"
	"gitlab.com/crankykernel/maker/go/config"
	"gitlab.com/crankykernel/maker/go/log"
	"math/rand"
	"net/http"
	"time"
)

type Authenticator struct {
	username string
	password string
}

func NewAuthenticator(configFilename string) *Authenticator {
	rand.Seed(time.Now().UnixNano())

	m := Authenticator{}

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
		b[i] = alphanumerics[rand.Intn(len(alphanumerics))]
	}
	return string(b)
}

// Middleware function, which will be called for each request
func (m *Authenticator) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if ok {
			if username == m.username {
				passwordOk, err := auth.CheckPassword(password, m.password)
				if err != nil {
					log.WithError(err).WithFields(log.Fields{
						"username": username,
					}).Errorf("An error occurred while checking password for user")
				} else if passwordOk {
					next.ServeHTTP(w, r)
					return
				}
			}
			log.WithField("username", username).Errorf("Login failed")
		}
		w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
		w.WriteHeader(http.StatusUnauthorized)
	})
}
