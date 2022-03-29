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
	"bytes"
	"encoding/json"
	"github.com/project-alvarium/scoring-apps-go/internal/config"
	"github.com/project-alvarium/scoring-apps-go/pkg/policies"
	"github.com/project-alvarium/scoring-apps-go/pkg/requests"
	"github.com/project-alvarium/scoring-apps-go/pkg/responses"
	"io/ioutil"
	"net/http"
)

type OpenPolicyProvider struct {
	cfg config.OpenPolicyConfig
}

func NewOpenPolicyProvider(cfg config.OpenPolicyConfig) PolicyProvider {
	p := OpenPolicyProvider{}
	p.cfg = cfg
	return &p
}

func (p *OpenPolicyProvider) GetWeights(classifier string) ([]policies.Weight, error) {

	// Send request
	url := p.cfg.Provider.Uri() + p.cfg.WeightsInfo.Path
	request := requests.OpaWeightsRequest{Classifier: classifier}
	b, err := json.Marshal(&request)

	if err != nil {
		return nil, err
	}
	result, err := http.Post(
		url,
		"application/json",
		bytes.NewBuffer(b),
	)
	if err != nil {
		return nil, err
	}

	// Read body
	b, err = ioutil.ReadAll(result.Body)
	if err != nil {
		return nil, err
	}
	result.Body.Close()
	// Unmarshal body and convert it to dcf weight array
	var response responses.OpaWeightsResponse
	err = json.Unmarshal(b, &response)
	if err != nil {
		return nil, err
	}
	var weights []policies.Weight
	for k, v := range response.Weights {
		weight := policies.Weight{}
		weight.AnnotationKey = k
		weight.Value = v
		weights = append(weights, weight)
	}
	return weights, nil
}
