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

package populator_api

import (
	"context"
	"github.com/gorilla/mux"
	SdkConfig "github.com/project-alvarium/alvarium-sdk-go/pkg/config"
	"github.com/project-alvarium/provider-logging/pkg/interfaces"
	"github.com/project-alvarium/provider-logging/pkg/logging"
	"github.com/project-alvarium/scoring-apps-go/internal/db"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// HttpServer contains references to dependencies required by the http server implementation.
type HttpServer struct {
	config  SdkConfig.ServiceInfo
	dbMongo *db.MongoProvider
	logger  interfaces.Logger
	router  *mux.Router
}

// NewHttpServer is a factory method that returns an initialized HttpServer receiver struct.
func NewHttpServer(router *mux.Router, config SdkConfig.ServiceInfo, dbMongo *db.MongoProvider, logger interfaces.Logger) *HttpServer {
	return &HttpServer{
		config:  config,
		dbMongo: dbMongo,
		logger:  logger,
		router:  router,
	}
}

// BootstrapHandler fulfills the BootstrapHandler contract.  It creates two go routines -- one that executes ListenAndServe()
// and another that waits on closure of a context's done channel before calling Shutdown() to cleanly shut down the
// http server.
func (b *HttpServer) BootstrapHandler(
	ctx context.Context,
	wg *sync.WaitGroup) bool {

	// this allows env override to explicitly set the value used
	// for ListenAndServe as needed for different deployments
	addr := ":" + strconv.Itoa(b.config.Port)

	timeout := time.Millisecond * 10000
	server := &http.Server{
		Addr:         addr,
		Handler:      b.router,
		WriteTimeout: timeout,
		ReadTimeout:  timeout,
	}

	b.logger.Write(logging.InfoLevel, "Web server starting ("+addr+")")

	wg.Add(1)
	go func() {
		defer wg.Done()

		_ = server.ListenAndServe()
		b.logger.Write(logging.InfoLevel, "Web server stopped")
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		<-ctx.Done()
		//DEBUG
		b.logger.Write(logging.InfoLevel, "Web server shutting down")
		_ = server.Shutdown(ctx)
		b.dbMongo.Close(ctx)
		//DEBUG
		b.logger.Write(logging.InfoLevel, "Web server shut down")
	}()

	return true
}
