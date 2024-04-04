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

package populator

import (
	"encoding/json"

	SdkConfig "github.com/project-alvarium/alvarium-sdk-go/pkg/config"
	"github.com/project-alvarium/scoring-apps-go/internal/config"
)

// ApplicationConfig serves as the root node for configuration and contains targeted child types with specialized
// concerns.
type ApplicationConfig struct {
	Databases []config.DatabaseInfo `json:"databases,omitempty"`
	Hash      SdkConfig.HashInfo    `json:"hash,omitempty"`
	Logging   SdkConfig.LoggingInfo `json:"logging,omitempty"`
}

func (a ApplicationConfig) AsString() string {
	b, _ := json.Marshal(a)
	return string(b)
}
