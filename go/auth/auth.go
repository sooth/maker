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

package auth

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"golang.org/x/crypto/argon2"
	"math/rand"
	"strings"
)

const (
	TYPE_ARGON2ID = "argon2id"
)

const PASSWORD_TYPE = TYPE_ARGON2ID

const SALT_SIZE = 16

func genSalt() ([]byte, error) {
	return genRandom(SALT_SIZE)
}

func genRandom(size int) ([]byte, error) {
	bytes := make([]byte, size)
	_, err := rand.Read(bytes)
	return bytes, err
}

func encode(input []byte) string {
	return hex.EncodeToString(input)
}

func decode(input string) ([]byte, error) {
	return hex.DecodeString(input)
}

func CheckPassword(password string, encodedPassword string) (bool, error) {
	passwordType, salt, passwordHash, err := DecodePassword(encodedPassword)
	if err != nil {
		return false, err
	}
	switch passwordType {
	case TYPE_ARGON2ID:
		hash := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)
		if bytes.Equal(hash, passwordHash) {
			return true, nil
		}
	}
	return false, nil
}

func DecodePassword(input string) (passwordType string, salt []byte, password []byte, err error) {
	parts := strings.Split(input, "$")
	if len(parts) != 3 {
		err = fmt.Errorf("invalid encoded password")
	} else {
		passwordType = parts[0]
		salt, err = decode(parts[1])
		password, err = decode(parts[2])
	}
	return passwordType, salt, password, err
}

func EncodePassword(password string) (string, error) {
	salt, err := genSalt()
	if err != nil {
		return "", err
	}
	encoded := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)
	return fmt.Sprintf("%s$%s$%s",
		TYPE_ARGON2ID, encode(salt), encode(encoded)), nil
}
