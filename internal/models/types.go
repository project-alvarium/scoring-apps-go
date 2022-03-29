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

package models

import (
	"github.com/oklog/ulid/v2"
	"github.com/project-alvarium/scoring-apps-go/pkg/responses"
	"time"
)

type MongoRecord struct {
	ObjectId     string    `bson:"_id,omitempty"`
	Description  string    `json:"description,omitempty"`
	Id           string    `json:"id,omitempty"`
	Seed         string    `json:"seed,omitempty"`
	Signature    string    `json:"signature,omitempty"`
	Timestamp    string    `json:"timestamp,omitempty"`
	TimestampISO time.Time `json:"timestampiso,omitempty"`
	Confidence   float64   `json:"confidence"`
}

// CopyForUpdate is necessary when updating a document in Mongo because the ObjectId on the incoming document must be
// blank. If not, there will be an error indicating a collision on the already populated, immutable value in the database.
func (mr MongoRecord) CopyForUpdate() MongoRecord {
	return MongoRecord{
		Description:  mr.Description,
		Id:           mr.Id,
		Seed:         mr.Seed,
		Signature:    mr.Signature,
		Timestamp:    mr.Timestamp,
		TimestampISO: mr.TimestampISO,
		Confidence:   mr.Confidence,
	}
}

func SampleFromMongoRecord(mr MongoRecord) responses.SampleData {
	parsed, _ := ulid.Parse(mr.Id)
	return responses.SampleData{
		Description: mr.Description,
		Id:          parsed,
		Seed:        mr.Seed,
		Signature:   mr.Signature,
		Timestamp:   mr.Timestamp,
	}
}

func ViewModelFromMongoRecord(mr MongoRecord) responses.DataViewModel {
	parsed, _ := ulid.Parse(mr.Id)
	vm := responses.DataViewModel{}
	vm.Confidence = mr.Confidence
	vm.Description = mr.Description
	vm.Id = parsed
	vm.Seed = mr.Seed
	vm.Signature = mr.Signature
	vm.Timestamp = mr.Timestamp

	return vm
}
