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

package calculator

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/project-alvarium/alvarium-sdk-go/pkg/interfaces"
)

// Writer can be used as an assistant for debugging so I'm going to leave it for now. If you're unsure whether a message
// is being delivered at some point of the internal handoff, plug in the Writer to have it log the relevant keys.
type Writer struct {
	chKeys chan string
	logger interfaces.Logger
}

func NewWriter(chKeys chan string, logger interfaces.Logger) Writer {
	return Writer{
		chKeys: chKeys,
		logger: logger,
	}
}

func (w *Writer) BootstrapHandler(ctx context.Context, wg *sync.WaitGroup) bool {
	wg.Add(1)
	go func() { // Process messages
		defer wg.Done()

		for {
			msg, ok := <-w.chKeys
			if !ok {
				return
			}

			w.logger.Write(slog.LevelDebug, fmt.Sprintf("key received %s", msg))
		}
	}()

	wg.Add(1)
	go func() { // Graceful shutdown
		defer wg.Done()

		<-ctx.Done()
		w.logger.Write(slog.LevelInfo, "shutdown received")
	}()
	return true
}
