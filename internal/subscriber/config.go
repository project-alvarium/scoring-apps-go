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

package subscriber

import (
	"encoding/json"
	sdkConfig "github.com/project-alvarium/alvarium-sdk-go/pkg/config"
	logging "github.com/project-alvarium/provider-logging/pkg/config"
	"github.com/project-alvarium/scoring-apps-go/internal/config"
)

type ApplicationConfig struct {
	Database config.DatabaseInfo `json:"database,omitempty"`
	Sdk      sdkConfig.SdkInfo   `json:"sdk,omitempty"`
	Stream   config.PubSubInfo   `json:"stream,omitempty"`
	Logging  logging.LoggingInfo `json:"logging,omitempty"`
	Key      string              `json:"preSharedKey,omitempty"` // Key is for IOTA support, shared key. Needs to be moved into SDK IotaStreamConfig
}

func (a ApplicationConfig) AsString() string {
	b, _ := json.Marshal(a)
	return string(b)
}
