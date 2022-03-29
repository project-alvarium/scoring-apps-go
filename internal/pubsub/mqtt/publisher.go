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

package mqtt

import (
	"context"
	"encoding/json"
	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/project-alvarium/alvarium-sdk-go/pkg/config"
	"github.com/project-alvarium/scoring-apps-go/internal/pubsub/interfaces"
	"github.com/project-alvarium/scoring-apps-go/pkg/msg"
	"time"
)

const (
	waitOnClose    uint          = 250
	publishTimeout time.Duration = 2000
)

type mqttPublisher struct {
	endpoint   config.MqttConfig
	mqttClient MQTT.Client
}

func NewMqttPublisher(cfg config.MqttConfig) interfaces.Publisher {
	opts := MQTT.NewClientOptions()
	opts.AddBroker(cfg.Provider.Uri())
	opts.SetClientID(cfg.ClientId)
	opts.SetUsername(cfg.User)
	opts.SetPassword(cfg.Password)
	opts.SetCleanSession(cfg.Cleanness)

	p := mqttPublisher{
		endpoint:   cfg,
		mqttClient: MQTT.NewClient(opts),
	}

	return &p
}

func (p *mqttPublisher) Publish(ctx context.Context, message msg.PublishWrapper) error {
	// Verify connectivity first. If it's been dropped, this will attempt one reconnect before publish
	err := p.reconnect()
	if err != nil {
		return err
	}

	b, _ := json.Marshal(message)
	// publish to all topics
	for _, topic := range p.endpoint.Topics {
		token := p.mqttClient.Publish(topic, byte(p.endpoint.Qos), false, b)
		token.WaitTimeout(time.Millisecond * publishTimeout)
	}
	return nil
}

func (p *mqttPublisher) Close() error {
	p.mqttClient.Disconnect(waitOnClose)
	return nil
}

func (p *mqttPublisher) reconnect() error {
	if !p.mqttClient.IsConnected() {
		token := p.mqttClient.Connect()
		if token.Wait() && token.Error() != nil {
			return token.Error()
		}
	}
	return nil
}
