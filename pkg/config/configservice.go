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

package config

import (
	"github.com/spf13/viper"
	"github.com/crankykernel/maker/pkg/log"
	"sync"
)

var subscribers map[chan bool]bool
var lock sync.RWMutex

func init() {
	subscribers = make(map[chan bool]bool)
}

func Subscribe() chan bool {
	lock.Lock()
	defer lock.Unlock()
	channel := make(chan bool)
	subscribers[channel] = true
	return channel
}

func Unsubscribe(channel chan bool) {
	lock.Lock()
	lock.Unlock()
	subscribers[channel] = false
	delete(subscribers, channel)
}

func WriteConfig() {
	viper.SetConfigFile("maker.yaml")
	log.Infof("Writing configuration file %s.", viper.ConfigFileUsed())
	viper.WriteConfig()
	lock.RLock()
	defer lock.RUnlock()
	for channel := range subscribers {
		channel <- true
	}
}

func Set(key string, val string) {
	lock.Lock()
	defer lock.Unlock()
	viper.Set(key, val)
}

func GetString(key string) string {
	return viper.GetString(key)
}