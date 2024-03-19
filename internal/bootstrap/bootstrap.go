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

/*
Package bootstrap contains all abstractions and implementation necessary to bootstrap the application.
*/
package bootstrap

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/project-alvarium/alvarium-sdk-go/pkg/config"
)

// Run is the bootstrap process entry point. All relevant application components can be initialized by providing a
// BootstrapHandler implementation to the handlers array.
func Run(
	ctx context.Context,
	cancel context.CancelFunc,
	configuration config.Configuration,
	handlers []BootstrapHandler) {

	wg, _ := initWaitGroup(ctx, cancel, configuration, handlers)

	wg.Wait()
}

func initWaitGroup(
	ctx context.Context,
	cancel context.CancelFunc,
	configuration config.Configuration,
	handlers []BootstrapHandler) (*sync.WaitGroup, bool) {

	startedSuccessfully := true

	var wg sync.WaitGroup
	// call individual bootstrap handlers.
	translateInterruptToCancel(ctx, &wg, cancel)
	for i := range handlers {
		if handlers[i](ctx, &wg) == false {
			cancel()
			startedSuccessfully = false
			break
		}
	}

	return &wg, startedSuccessfully
}

// translateInterruptToCancel spawns a go routine to translate the receipt of a SIGTERM signal to a call to cancel
// the context used by the bootstrap implementation.
func translateInterruptToCancel(ctx context.Context, wg *sync.WaitGroup, cancel context.CancelFunc) {
	wg.Add(1)
	go func() {
		defer wg.Done()

		signalStream := make(chan os.Signal, 1)
		defer func() {
			signal.Stop(signalStream)
			close(signalStream)
		}()
		signal.Notify(signalStream, os.Interrupt, syscall.SIGTERM)
		select {
		case <-signalStream:
			cancel()
			return
		case <-ctx.Done():
			return
		}
	}()
}
