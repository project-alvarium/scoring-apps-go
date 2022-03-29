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

package config

import (
	"encoding/json"
	"fmt"
	"github.com/project-alvarium/alvarium-sdk-go/pkg/config"
	"github.com/project-alvarium/alvarium-sdk-go/pkg/contracts"
	"github.com/project-alvarium/scoring-apps-go/pkg/policies"
)

type DatabaseType string

const (
	DBArango DatabaseType = "arango"
	DBMongo  DatabaseType = "mongo"
)

func (t DatabaseType) Validate() bool {
	if t == DBArango || t == DBMongo {
		return true
	}
	return false
}

type PolicyType string

const (
	OpenPolicy  PolicyType = "opa"
	LocalPolicy PolicyType = "local"
)

func (t PolicyType) Validate() bool {
	if t == OpenPolicy || t == LocalPolicy {
		return true
	}
	return false
}

type ArangoConfig struct {
	DatabaseName string             `json:"databaseName,omitempty"`
	Edges        []EdgeInfo         `json:"edges,omitempty"`
	GraphName    string             `json:"graphName,omitempty"`
	Provider     config.ServiceInfo `json:"provider,omitempty"`
	Vertexes     []string           `json:"vertexes,omitempty"` // Vertexes only require the relevant collection names
}

// MongoConfig provides configuration attributes relative to a MongoDB connection
type MongoConfig struct {
	Host       string `json:"host,omitempty"`
	Port       int    `json:"port,omitempty"`
	Username   string `json:"username,omitempty"`
	Password   string `json:"password,omitempty"`
	Collection string `json:"collection,omitempty"`
	DbName     string `json:"dbName,omitempty"`
}

type EdgeInfo struct {
	CollectionName string   `json:"collectionName,omitempty"`
	From           []string `json:"from,omitempty"`
	To             []string `json:"to,omitempty"`
}

type DatabaseInfo struct {
	Type   DatabaseType `json:"type,omitempty"`
	Config interface{}  `json:"config,omitempty"`
}

func (d *DatabaseInfo) UnmarshalJSON(data []byte) (err error) {
	type Alias struct {
		Type DatabaseType
	}
	a := Alias{}
	if err = json.Unmarshal(data, &a); err != nil {
		return err
	}
	if !a.Type.Validate() {
		return fmt.Errorf("invalid DatabaseType value provided %s", a.Type)
	}
	if a.Type == DBArango {
		type arangoAlias struct {
			Type   DatabaseType `json:"type,omitempty"`
			Config ArangoConfig `json:"config,omitempty"`
		}
		i := arangoAlias{}
		// Error with unmarshaling
		if err = json.Unmarshal(data, &i); err != nil {
			return err
		}
		d.Type = i.Type
		d.Config = i.Config
	} else if a.Type == DBMongo {
		type mongoAlias struct {
			Type   DatabaseType `json:"type,omitempty"`
			Config MongoConfig  `json:"config,omitempty"`
		}
		i := mongoAlias{}
		// Error with unmarshaling
		if err = json.Unmarshal(data, &i); err != nil {
			return err
		}
		d.Type = i.Type
		d.Config = i.Config
	}
	return nil
}

type PolicyInfo struct {
	Type   PolicyType  `json:"type,omitempty"`
	Config interface{} `json:"config,omitempty"`
}

type OpenPolicyConfig struct {
	Provider    config.ServiceInfo `json:"provider,omitempty"`
	WeightsInfo OpaWeightsInfo     `json:"weights,omitempty"`
}

type OpaWeightsInfo struct {
	Path string `json:"path,omitempty"`
}

type LocalPolicyConfig struct {
	WeightsInfo []policies.DcfPolicy `json:"weights,omitempty"`
}

func (p *PolicyInfo) UnmarshalJSON(data []byte) (err error) {
	type Alias struct {
		Type PolicyType
	}
	a := Alias{}
	if err = json.Unmarshal(data, &a); err != nil {
		return err
	}
	if !a.Type.Validate() {
		return fmt.Errorf("invalid PolicyType value provided %s", a.Type)
	}
	if a.Type == OpenPolicy {
		type opaAlias struct {
			Type   PolicyType       `json:"type,omitempty"`
			Config OpenPolicyConfig `json:"config,omitempty"`
		}
		i := opaAlias{}
		if err = json.Unmarshal(data, &i); err != nil {
			return err
		}
		p.Type = i.Type
		p.Config = i.Config
	} else if a.Type == LocalPolicy {
		type localAlias struct {
			Type   PolicyType        `json:"type,omitempty"`
			Config LocalPolicyConfig `json:"config,omitempty"`
		}
		i := localAlias{}
		if err = json.Unmarshal(data, &i); err != nil {
			return err
		}
		p.Type = i.Type
		p.Config = i.Config
	} else {
		return fmt.Errorf("unhandled PolicyInfo.Type value %s", a.Type)
	}
	return nil
}

func (p *LocalPolicyConfig) UnmarshalJSON(data []byte) (err error) {
	type alias struct {
		WeightsInfo []policies.DcfPolicy `json:"weights,omitempty"`
	}
	a := alias{}

	err = json.Unmarshal(data, &a)
	if err != nil {
		return err
	}

	// Validate all annotation types loaded from the config
	for _, info := range a.WeightsInfo {
		for _, weight := range info.Weights {
			key := contracts.AnnotationType(weight.AnnotationKey)
			if !key.Validate() {
				return fmt.Errorf("invalid AnnotatorType value provided %s", key)
			}
		}
	}

	p.WeightsInfo = a.WeightsInfo
	return nil
}

// PubSubInfo encapsulates endpoint definitions for publishing and subscribing to the relevant platform providers.
type PubSubInfo struct {
	Publish   config.StreamInfo `json:"publisher,omitempty"`  //Defines the publisher endpoint
	Subscribe config.StreamInfo `json:"subscriber,omitempty"` //Defines the subscriber endpoint
}
