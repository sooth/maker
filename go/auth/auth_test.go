package auth

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestArgon2(t *testing.T) {
	assert := assert.New(t)

	encoded, err := EncodePassword("password")
	assert.Nil(err)

	passwordType, salt, password, err := DecodePassword(encoded)
	assert.Nil(err)
	assert.Equal(PASSWORD_TYPE, passwordType)
	assert.NotEmpty(salt)
	assert.NotEmpty(password)

	ok, err := CheckPassword("password", encoded)
	assert.Nil(err)
	assert.True(ok)

	ok, err = CheckPassword("password1", encoded)
	assert.Nil(err)
	assert.False(ok)
}
