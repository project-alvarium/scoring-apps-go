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
	"log/slog"

	sdkConfig "github.com/project-alvarium/alvarium-sdk-go/pkg/config"
	"github.com/project-alvarium/alvarium-sdk-go/pkg/factories"
	"github.com/project-alvarium/scoring-apps-go/internal/bootstrap"
	"github.com/project-alvarium/scoring-apps-go/internal/calculator"
	"github.com/project-alvarium/scoring-apps-go/internal/calculator/policy"
	"github.com/project-alvarium/scoring-apps-go/internal/config"
	"github.com/project-alvarium/scoring-apps-go/pkg/policies"
	"os"
)

func main() {
	// Resolve mode
	var mode string
	flag.StringVar(&mode,
		"mode",
		"default",
		"The policy mode of operation for the application.")

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
		tmpLog := factories.NewLogger(sdkConfig.LoggingInfo{MinLogLevel: slog.LevelError})
		tmpLog.Error(err.Error())
		os.Exit(1)
	}

	cfg := calculator.ApplicationConfig{}
	err = reader.Read(configPath, &cfg)
	if err != nil {
		tmpLog := factories.NewLogger(sdkConfig.LoggingInfo{MinLogLevel: slog.LevelError})
		tmpLog.Error(err.Error())
		os.Exit(1)
	}

	logger := factories.NewLogger(cfg.Logging)
	logger.Write(slog.LevelDebug, "config loaded successfully")
	logger.Write(slog.LevelDebug, cfg.AsString())

	chKeys := make(chan string)
	sub, err := calculator.NewSubscriber(cfg.Stream.Subscribe, chKeys, logger)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	chScore := make(chan string)
	coll := calculator.NewCollector(chKeys, chScore, logger)

	p := policies.DcfPolicy{}
	p.Name = mode
	provider, err := policy.NewPolicyProvider(cfg.Policy, logger)
	if err != nil {
		logger.Error(err.Error())
		return
	}
	weights, err := provider.GetWeights(p.Name)
	if err != nil {
		logger.Error(err.Error())
		return
	}
	p.Weights = weights
	calc := calculator.NewCalculator(chScore, cfg.Database, logger, p)
	ctx, cancel := context.WithCancel(context.Background())
	bootstrap.Run(
		ctx,
		cancel,
		cfg,
		[]bootstrap.BootstrapHandler{
			sub.BootstrapHandler,
			coll.BootstrapHandler,
			calc.BootstrapHandler,
		})
}
