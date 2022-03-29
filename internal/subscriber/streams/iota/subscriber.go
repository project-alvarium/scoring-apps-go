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

package iota

/*
#cgo CFLAGS: -I./include -DIOTA_STREAMS_CHANNELS_CLIENT
#cgo LDFLAGS: -L./include -liota_streams_c
#include <channels.h>
*/
import "C"
import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/project-alvarium/alvarium-sdk-go/pkg/config"
	"github.com/project-alvarium/alvarium-sdk-go/pkg/message"
	logInterface "github.com/project-alvarium/provider-logging/pkg/interfaces"
	"github.com/project-alvarium/provider-logging/pkg/logging"
	"github.com/project-alvarium/scoring-apps-go/internal/subscriber"
	"io/ioutil"
	"math/rand"
	"net/http"
	"sync"
	"time"
	"unsafe"
)

/* The Subscriber has been implemented inline here because I'm not thinking favorably about putting stream subscription
   into the SDK. The SDK interface should be kept simple, and its responsibility limited to Annotations.

	However it's conceivable we might have a module -- provider-streams -- that would contain this inline IOTA integration
    as well as other streaming platforms like MQTT, Kafka, etc.

	I used the Iota Publisher inside the SDK and also the RustAuthorConsole as examples informing this work.
*/

// For randomized seed generation
const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
const payloadLength = 1024

type iotaSubscriber struct {
	cfg        config.IotaStreamConfig
	chPub      chan message.SubscribeWrapper
	logger     logInterface.Logger
	keyload    *C.message_links_t // The Keyload indicates a key needed by the publisher to send messages to the stream
	subscriber *C.subscriber_t    // The publisher is actually subscribed to the stream
	seed       string
	key        string
}

func NewIotaSubscriber(cfg config.IotaStreamConfig, pub chan message.SubscribeWrapper, logger logInterface.Logger, key string) subscriber.Subscriber {
	bytes := make([]byte, 64)
	rand.Seed(time.Now().UnixNano())
	for i := range bytes {
		bytes[i] = letterBytes[rand.Intn(len(letterBytes))]
	}

	seed := string(bytes)
	logger.Write(logging.DebugLevel, fmt.Sprintf("generated streams seed %s", seed))
	return &iotaSubscriber{
		cfg:    cfg,
		chPub:  pub,
		logger: logger,
		seed:   seed,
		key:    key,
	}
}

func (s *iotaSubscriber) Subscribe(ctx context.Context, wg *sync.WaitGroup) bool {
	err := s.connect()
	if err != nil {
		s.logger.Error(err.Error())
		return false
	}

	cancelled := false
	chRawOut := make(chan []byte)
	wg.Add(1)
	go func() {
		defer wg.Done()

		for !cancelled {
			time.Sleep(100 * time.Millisecond)
			err := s.Read(chRawOut)
			if err != nil {
				s.logger.Error(err.Error())
			}
		}
	}()

	wg.Add(1)
	go func(chBytes chan []byte) {
		defer wg.Done()
		for {
			b, ok := <-chBytes
			if !ok {
				return
			}
			var wrapped message.SubscribeWrapper
			err := json.Unmarshal(b, &wrapped)
			if err != nil {
				s.logger.Error(err.Error())
			} else {
				s.chPub <- wrapped
			}
		}
	}(chRawOut)

	wg.Add(1)
	go func() { // Graceful shutdown
		defer wg.Done()

		<-ctx.Done()
		s.logger.Write(logging.InfoLevel, "shutdown received")
		cancelled = true
		close(chRawOut)
	}()
	return true
}

func (s *iotaSubscriber) connect() error {
	// Generate Transport client
	transport := C.transport_client_new_from_url(C.CString(s.cfg.TangleNode.Uri()))
	s.logger.Write(logging.DebugLevel, fmt.Sprintf("transport established %s", s.cfg.TangleNode.Uri()))

	// Generate Subscriber instance
	cErr := C.sub_new(&s.subscriber, C.CString(s.seed), C.CString(s.cfg.Encoding), payloadLength, transport)
	s.logger.Write(logging.DebugLevel, fmt.Sprintf(get_error(cErr)))
	s.logger.Write(logging.DebugLevel, fmt.Sprintf("subscriber established seed=%s", s.seed))

	// Process announcement message
	rawId, err := s.getAnnouncementId(s.cfg.Provider.Uri())
	s.logger.Write(logging.DebugLevel, fmt.Sprintf("Got announcement"))
	if err != nil {
		return err
	}

	var pskid *C.psk_id_t
	// Store psk
	cErr = C.sub_store_psk(&pskid, s.subscriber, C.CString(s.key))
	s.logger.Write(logging.DebugLevel, fmt.Sprintf(get_error(cErr)))
	if cErr == C.ERR_OK {
		address := C.address_from_string(C.CString(rawId))
		cErr = C.sub_receive_announce(s.subscriber, address)
		s.logger.Write(logging.DebugLevel, fmt.Sprintf(get_error(cErr)))
		if cErr == C.ERR_OK {
			// Fetch sub link and pk for subscription
			var subLink *C.address_t
			var subPk *C.public_key_t

			cErr = C.sub_send_subscribe(&subLink, s.subscriber, address)
			s.logger.Write(logging.DebugLevel, fmt.Sprintf(get_error(cErr)))
			if cErr == C.ERR_OK {
				cErr = C.sub_get_public_key(&subPk, s.subscriber)
				s.logger.Write(logging.DebugLevel, fmt.Sprintf(get_error(cErr)))
				if cErr == C.ERR_OK {
					subIdStr := C.get_address_id_str(subLink)
					subPkStr := C.public_key_to_string(subPk)

					s.logger.Write(logging.DebugLevel, fmt.Sprintf("send subscription request %s", C.GoString(subIdStr)))
					r := subscriptionRequest{
						MsgId: C.GoString(subIdStr),
						Pk:    C.GoString(subPkStr),
					}
					body, _ := json.Marshal(&r)
					sendSubscriptionIdToAuthor(s.cfg.Provider.Uri(), body)
					s.logger.Write(logging.DebugLevel, "subscription request sent")

					// Free generated c strings from mem
					C.drop_str(subIdStr)
					C.drop_str(subPkStr)
					return nil
				}
			}
		}
	}
	return errors.New("failed to connect publisher")
}

func (s *iotaSubscriber) Read(chRawOut chan []byte) error {
	var messages *C.unwrapped_messages_t
	cErr := C.sub_sync_state(&messages, s.subscriber)
	//defer C.drop_unwrapped_messages(messages)

	if cErr == C.ERR_OK {
		count := int(C.get_payloads_count(messages))
		idx := 0
		for idx < count {
			msg := C.get_indexed_payload(messages, C.size_t(idx))
			out := C.GoBytes(unsafe.Pointer(msg.masked_payload.ptr), C.int(msg.masked_payload.size))
			fmt.Println(msg.masked_payload.ptr, msg.masked_payload.size)
			content := string(out)
			fmt.Println(fmt.Sprintf("Message -- len:%v txt:%s", len(out), content))
			if len(out) > 0 { // Sometimes empty messages come across during connect handshake
				chRawOut <- out
			}
			C.drop_payloads(msg)
			idx++
		}
	} else {
		return errors.New(get_error(cErr))
	}
	return nil
}

func (s *iotaSubscriber) Close() {
	C.sub_drop(s.subscriber)
}

func (s *iotaSubscriber) getAnnouncementId(url string) (string, error) {
	type announcementResponse struct {
		AnnouncementId string `json:"announcement_id"`
	}

	s.logger.Write(logging.DebugLevel, fmt.Sprintf("GET %s/get_announcement_id", url))
	resp, err := http.Get(url + "/get_announcement_id")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	s.logger.Write(logging.DebugLevel, fmt.Sprintf("announcement response - %s", string(bodyBytes)))
	var annResp announcementResponse
	if err := json.Unmarshal(bodyBytes, &annResp); err != nil {
		return "", err
	}
	return annResp.AnnouncementId, nil
}

func sendSubscriptionIdToAuthor(url string, body []byte) error {
	client := http.Client{}
	data := bytes.NewReader(body)
	req, err := http.NewRequest("POST", url+"/subscribe", data)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

type subscriptionRequest struct {
	MsgId string `json:"msgid"`
	Pk    string `json:"pk"`
}

func get_error(err C.err_t) string {
	var e = "Unknown Error"
	switch err {
	case C.ERR_OK:
		e = "Operation completed successfully"
	case C.ERR_OPERATION_FAILED:
		e = "Streams operation failed to complete successfully"
	case C.ERR_NULL_ARGUMENT:
		e = "The function was passed a null argument"
	case C.ERR_BAD_ARGUMENT:
		e = "The function was passed a bad argument"
	}
	return e
}
