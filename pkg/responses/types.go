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

package responses

import (
	"encoding/json"
	"github.com/oklog/ulid/v2"
	"github.com/project-alvarium/scoring-apps-go/pkg/documents"
)

type OpaWeightsResponse struct {
	Weights map[string]int `json:"result,omitempty"`
}

func (p *OpaWeightsResponse) UnmarshalJSON(data []byte) error {
	type alias struct {
		Result []map[string]int `json:"result,omitempty"`
	}

	a := alias{}

	err := json.Unmarshal(data, &a)
	if err != nil {
		return err
	}

	p.Weights = a.Result[0]
	return nil
}

type AnnotationListResponse struct {
	Count       int                    `json:"count"`
	Annotations []documents.Annotation `json:"annotations"`
}

type DocumentCountResponse struct {
	Count int `json:"count"`
}

// SampleData represents the data at play in the application's data path. We should not expect this type to have fields
// specific to the view model that includes data confidence. This type representation is also what is used to create the
// original hash, so we must map from view model to original data type. This further implies a shared library of "contract"
// data types available to the developer creating the view model, as well as the Alvarium SDK.
type SampleData struct {
	Description string    `json:"description,omitempty"`
	Id          ulid.ULID `json:"id,omitempty"`
	Seed        string    `json:"seed,omitempty"`
	Signature   string    `json:"signature,omitempty"`
	Timestamp   string    `json:"timestamp,omitempty"`
}

type DataViewModel struct {
	SampleData
	Confidence float64 `json:"confidence"`
}

type DataListResponse struct {
	Count     int             `json:"count"`               // Count is the number of items in the list.
	Documents []DataViewModel `json:"documents,omitempty"` // Documents is an array of the returned view models
}
