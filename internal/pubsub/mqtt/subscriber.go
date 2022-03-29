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
	"errors"
	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/project-alvarium/alvarium-sdk-go/pkg/config"
	"github.com/project-alvarium/scoring-apps-go/internal/pubsub/interfaces"
	"github.com/project-alvarium/scoring-apps-go/pkg/msg"
	"sync"
)

type mqttSubscriber struct {
	chPub      chan<- msg.SubscribeWrapper
	endpoint   config.MqttConfig
	mqttClient MQTT.Client
}

func NewMqttSubscriber(cfg config.MqttConfig) (interfaces.Subscriber, error) {
	// create MQTT options
	opts := MQTT.NewClientOptions()
	opts.AddBroker(cfg.Provider.Uri())
	opts.SetClientID(cfg.ClientId)
	opts.SetUsername(cfg.User)
	opts.SetPassword(cfg.Password)
	opts.SetCleanSession(cfg.Cleanness)

	var subscriber = mqttSubscriber{
		endpoint:   cfg,
		mqttClient: MQTT.NewClient(opts),
	}
	// no error to report
	return &subscriber, nil
}

func (s *mqttSubscriber) Subscribe(ctx context.Context, chMessage chan<- msg.SubscribeWrapper, chErrors chan<- error) {
	err := s.reconnect()
	if err != nil {
		chErrors <- err
		return
	}

	s.chPub = chMessage
	if len(s.endpoint.Topics) > 1 {
		topicsMap := make(map[string]byte)
		// build topic qos map
		for _, topic := range s.endpoint.Topics {
			topicsMap[topic] = byte(s.endpoint.Qos)
		}

		if token := s.mqttClient.SubscribeMultiple(topicsMap, s.mqttMessageHandler); token.Wait() {
			if token.Error() != nil {
				chErrors <- err
				return
			}
		}
	} else if len(s.endpoint.Topics) == 1 {
		var topic = s.endpoint.Topics[0]
		if token := s.mqttClient.Subscribe(topic, byte(s.endpoint.Qos), s.mqttMessageHandler); token.Wait() {
			if token.Error() != nil {
				chErrors <- err
				return
			}
		}
	} else {
		chErrors <- errors.New("at least one topic value should be configured")
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() { // Graceful shutdown
		defer wg.Done()

		<-ctx.Done()
		close(chMessage)
	}()
	wg.Wait()
}

func (s *mqttSubscriber) Close() error {
	s.mqttClient.Disconnect(waitOnClose)
	return nil
}

func (s *mqttSubscriber) reconnect() error {
	// Connect client to broker if not already connected
	if !s.mqttClient.IsConnected() {
		if token := s.mqttClient.Connect(); token.Wait() && token.Error() != nil {
			// Connecting to publisher
			return token.Error()
		}
	}
	return nil
}

// General message handing func
func (s *mqttSubscriber) mqttMessageHandler(client MQTT.Client, mqttMsg MQTT.Message) {
	var wrap msg.SubscribeWrapper
	json.Unmarshal(mqttMsg.Payload(), &wrap)
	s.chPub <- wrap
}
