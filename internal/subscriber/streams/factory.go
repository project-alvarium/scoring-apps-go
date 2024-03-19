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

package streams

import (
	"errors"
	"fmt"

	"github.com/project-alvarium/alvarium-sdk-go/pkg/config"
	"github.com/project-alvarium/alvarium-sdk-go/pkg/contracts"
	"github.com/project-alvarium/alvarium-sdk-go/pkg/interfaces"
	"github.com/project-alvarium/alvarium-sdk-go/pkg/message"
	"github.com/project-alvarium/scoring-apps-go/internal/subscriber"
	"github.com/project-alvarium/scoring-apps-go/internal/subscriber/streams/mqtt"
)

func NewSubscriber(cfg config.StreamInfo, pub chan message.SubscribeWrapper, key string, logger interfaces.Logger) (subscriber.Subscriber, error) {
	var sub subscriber.Subscriber

	switch cfg.Type {
	case contracts.MqttStream:
		endpoint, ok := cfg.Config.(config.MqttConfig)
		if !ok {
			return nil, errors.New("unknown type cast to MqttConfig failed")
		}
		sub = mqtt.NewMqttSubscriber(endpoint, pub, logger)
	default:
		return nil, errors.New(fmt.Sprintf("unrecognized stream provider type %s", cfg.Type))
	}
	return sub, nil
}
