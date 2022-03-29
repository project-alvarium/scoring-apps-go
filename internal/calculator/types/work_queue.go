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

import "sync"

type WorkQueue struct {
	items   []string
	mutex   sync.Mutex
	Workers *Workers
}

func NewWorkQueue() *WorkQueue {
	wq := WorkQueue{}
	wq.items = []string{}
	wq.Workers = NewWorkers()
	return &wq
}

func (wq *WorkQueue) Append(v string) {
	wq.mutex.Lock()
	defer wq.mutex.Unlock()
	wq.items = append(wq.items, v)
}

func (wq *WorkQueue) Len() int {
	wq.mutex.Lock()
	defer wq.mutex.Unlock()
	return len(wq.items)
}

func (wq *WorkQueue) First() string {
	wq.mutex.Lock()
	defer wq.mutex.Unlock()
	if len(wq.items) > 0 {
		t := wq.items[0]
		copy(wq.items[0:], wq.items[1:])
		wq.items = wq.items[:len(wq.items)-1]
		return t
	}
	return ""
}

type Workers struct {
	count int
	mutex sync.Mutex
}

func NewWorkers() *Workers {
	return &Workers{}
}

func (w *Workers) Increment() {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	w.count++
}

func (w *Workers) Decrement() {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	w.count--
}

func (w *Workers) Count() int {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	return w.count
}
