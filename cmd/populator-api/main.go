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

package main

import (
	"context"
	"flag"
	"github.com/gorilla/mux"
	logConfig "github.com/project-alvarium/provider-logging/pkg/config"
	logFactory "github.com/project-alvarium/provider-logging/pkg/factories"
	"github.com/project-alvarium/provider-logging/pkg/logging"
	"github.com/project-alvarium/scoring-apps-go/internal/bootstrap"
	"github.com/project-alvarium/scoring-apps-go/internal/config"
	"github.com/project-alvarium/scoring-apps-go/internal/db"
	populator_api "github.com/project-alvarium/scoring-apps-go/internal/populator-api"
	"os"
)

func main() {
	// Load config
	var configPath string
	flag.StringVar(&configPath,
		"cfg",
		"./res/config.json",
		"Path to JSON configuration file.")
	flag.Parse()

	fileFormat := config.GetFileExtension(configPath)
	reader, err := config.NewReader(fileFormat)
	if err != nil {
		tmpLog := logFactory.NewLogger(logConfig.LoggingInfo{MinLogLevel: logging.ErrorLevel})
		tmpLog.Error(err.Error())
		os.Exit(1)
	}

	cfg := populator_api.ApplicationConfig{}
	err = reader.Read(configPath, &cfg)
	if err != nil {
		tmpLog := logFactory.NewLogger(logConfig.LoggingInfo{MinLogLevel: logging.ErrorLevel})
		tmpLog.Error(err.Error())
		os.Exit(1)
	}

	logger := logFactory.NewLogger(cfg.Logging)
	logger.Write(logging.DebugLevel, "config loaded successfully")
	logger.Write(logging.DebugLevel, cfg.AsString())

	dbMongo, err := db.NewMongoProvider(cfg.Databases, logger)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(-1)
	}

	dbArango, err := db.NewArangoClient(cfg.Databases, logger)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(-1)
	}

	r := mux.NewRouter()
	populator_api.LoadRestRoutes(r, dbArango, dbMongo, logger)
	ctx, cancel := context.WithCancel(context.Background())
	bootstrap.Run(
		ctx,
		cancel,
		cfg,
		[]bootstrap.BootstrapHandler{
			populator_api.NewHttpServer(r, cfg.Endpoint, dbMongo, logger).BootstrapHandler,
		})
}
