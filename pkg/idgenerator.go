// Copyright (C) 2018 Cranky Kernel
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

package pkg

import (
	"math/rand"
	"time"
	"github.com/oklog/ulid"
)

type IdGenerator struct {
	entropy *rand.Rand
}

func NewIdGenerator() *IdGenerator {
	return &IdGenerator{
		entropy: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (g *IdGenerator) GetID(timestamp *time.Time) (ulid.ULID, error) {
	if timestamp == nil {
		_timestamp := time.Now()
		timestamp = &_timestamp
	}
	return ulid.New(ulid.Timestamp(*timestamp), g.entropy)
}
