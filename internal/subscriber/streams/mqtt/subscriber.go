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
	"log/slog"
	"os"
	"sync"
	"time"

	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/project-alvarium/alvarium-sdk-go/pkg/config"
	"github.com/project-alvarium/alvarium-sdk-go/pkg/interfaces"
	"github.com/project-alvarium/alvarium-sdk-go/pkg/message"
	"github.com/project-alvarium/scoring-apps-go/internal/subscriber"
)

type mqttSubscriber struct {
	chPub      chan message.SubscribeWrapper
	endpoint   config.MqttConfig
	logger     interfaces.Logger
	mqttClient MQTT.Client
}

func NewMqttSubscriber(endpoint config.MqttConfig, pub chan message.SubscribeWrapper, logger interfaces.Logger) subscriber.Subscriber {
	// create MQTT options
	opts := MQTT.NewClientOptions()
	opts.AddBroker(endpoint.Provider.Uri())
	opts.SetClientID(endpoint.ClientId)
	opts.SetUsername(endpoint.User)
	opts.SetPassword(endpoint.Password)
	opts.SetCleanSession(endpoint.Cleanness)

	var subscriber = mqttSubscriber{
		chPub:      pub,
		endpoint:   endpoint,
		logger:     logger,
		mqttClient: MQTT.NewClient(opts),
	}
	// no error to report
	return &subscriber
}

func (s *mqttSubscriber) Subscribe(ctx context.Context, wg *sync.WaitGroup) bool {
	err := s.reconnect()
	if err != nil {
		s.logger.Error(err.Error())
		return false
	}

	if len(s.endpoint.Topics) > 1 {
		topicsMap := make(map[string]byte)
		// build topic qos map
		for _, topic := range s.endpoint.Topics {
			topicsMap[topic] = byte(s.endpoint.Qos)
		}

		if token := s.mqttClient.SubscribeMultiple(topicsMap, s.mqttMessageHandler); token.Wait() {
			if token.Error() != nil {
				s.logger.Error(token.Error().Error())
				return false
			} else {
				s.logger.Write(slog.LevelDebug, "successfully subscribed (multiple)")
			}
		}
	} else if len(s.endpoint.Topics) == 1 {
		var topic = s.endpoint.Topics[0]
		if token := s.mqttClient.Subscribe(topic, byte(s.endpoint.Qos), s.mqttMessageHandler); token.Wait() {
			if token.Error() != nil {
				s.logger.Error(token.Error().Error())
				return false
			} else {
				s.logger.Write(slog.LevelDebug, "successfully subscribed")
			}
		}
	} else {
		s.logger.Error("at least one topic value should be configured")
		return false
	}

	wg.Add(1)
	go func() { // Graceful shutdown
		defer wg.Done()

		<-ctx.Done()
		close(s.chPub)
		s.logger.Write(slog.LevelInfo, "shutdown received")
	}()
	return true
}

func (s *mqttSubscriber) Close() {
	s.mqttClient.Disconnect(1000)
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

// utility function to monitor OS signals
func (s *mqttSubscriber) monitorSignals(signals chan os.Signal, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()
	// Looping for system interrupts then closing client
	for {
		select {
		case <-signals:
			s.logger.Write(slog.LevelInfo, "Interrupt is detected. Message Consumption stopped.")
			s.Close()
			break
		}
		time.Sleep(time.Millisecond * 100)
	}
}

// General message handing func
func (s *mqttSubscriber) mqttMessageHandler(client MQTT.Client, mqttMsg MQTT.Message) {
	var wrapped message.SubscribeWrapper
	err := json.Unmarshal(mqttMsg.Payload(), &wrapped)
	if err != nil {
		s.logger.Error(err.Error())
	} else {
		s.chPub <- wrapped
	}
}
