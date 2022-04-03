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

package policy

import (
	"errors"
	"fmt"
	"github.com/project-alvarium/provider-logging/pkg/logging"

	"github.com/project-alvarium/provider-logging/pkg/interfaces"
	"github.com/project-alvarium/scoring-apps-go/internal/config"
)

func NewPolicyProvider(policyInfo config.PolicyInfo, logger interfaces.Logger) (PolicyProvider, error) {
	switch policyInfo.Type {
	case config.LocalPolicy:
		cfg, ok := policyInfo.Config.(config.LocalPolicyConfig)
		if !ok {
			return nil, errors.New("invalid cast for local policy config")
		}
		return NewLocalPolicyProvider(cfg), nil

	case config.OpenPolicy:
		cfg, ok := policyInfo.Config.(config.OpenPolicyConfig)
		if !ok {
			return nil, errors.New("invalid cast type for OpenPolicyConfig")
		} else {
			logger.Write(logging.DebugLevel, "OPA connection successful")
		}
		return NewOpenPolicyProvider(cfg), nil

	default:
		return nil, fmt.Errorf("Unrecognized config value %s", policyInfo.Type)
	}

}
