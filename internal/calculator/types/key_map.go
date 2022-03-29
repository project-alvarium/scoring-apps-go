/*******************************************************************************
 * Copyright 2022 Dell Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software distributed under the License
 * is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express
 * or implied. See the License for the specific language governing permissions and limitations under
 * the License.
 *******************************************************************************/

package types

import (
	"sync"
	"time"
)

// KeyMap is responsible for managing the list of keys for which we need to calculate scores.
type KeyMap struct {
	items map[string]time.Time
	mutex sync.Mutex
}

func NewKeyMap() *KeyMap {
	km := KeyMap{}
	km.items = make(map[string]time.Time)
	return &km
}

// Add will add a key to the map if it doesn't already exist. If it does exist, it will update the timestamp.
func (km *KeyMap) Add(key string) {
	km.mutex.Lock()
	defer km.mutex.Unlock()

	// Either way -- adding or updating -- the assignment is the same.
	km.items[key] = time.Now()
}

// Poll will return all of the keys that are ready for processing. This also removes the keys from the internal map.
func (km *KeyMap) Poll(interval int64) []string {
	km.mutex.Lock()
	defer km.mutex.Unlock()

	// find all relevant items
	var found []string
	for k, v := range km.items {
		if time.Now().Sub(v).Milliseconds() >= interval {
			found = append(found, k)
		}
	}

	// now delete all relevant items (can't delete during range operation)
	for _, k := range found {
		delete(km.items, k)
	}
	return found
}

// Conceivably at some point there be a Store() method for dealing with in-flight keys if the service gets shut down.
// Those will simply get dropped for now.
