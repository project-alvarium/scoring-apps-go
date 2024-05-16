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

package documents

import (
	"math"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/project-alvarium/alvarium-sdk-go/pkg/contracts"
	"github.com/project-alvarium/scoring-apps-go/pkg/policies"
)

const (
	EdgeLineage       string = "lineage"
	EdgeScoring       string = "scoring"
	EdgeTrust         string = "trust"
	EdgeStack         string = "stack"
	VertexAnnotations string = "annotations"
	VertexData        string = "data"
	VertexScores      string = "scores"
)

// Data represents a document in the "data" vertex collection
type Data struct {
	Key       string    `json:"_key,omitempty"`      // Key uniquely identifies the document in the database
	Timestamp time.Time `json:"timestamp,omitempty"` // Timestamp indicates when the document was created
}

// Annotation represents a document in the "annotation" vertex collection
type Annotation struct {
	Key         string              `json:"_key,omitempty"`    // Key uniquely identifies the document in the database
	DataRef     string              `json:"dataRef,omitempty"` // DataRef points to the key of the data being annotated
	Hash        contracts.HashType  `json:"hash,omitempty"`    // Hash identifies which algorithm was used to construct the hash
	Host        string              `json:"host,omitempty"`    // Host is the hostname of the node making the annotation
	Tag         string              `json:"tag,omitempty"`     // Tag is the hash of the source artifact tag that emitted data being annotated
	Layer       contracts.LayerType `json:"layer,omitempty"`
	Kind        string              `json:"type,omitempty"`      // Kind indicates what kind of annotation this is. Defined as string to allow for annotation types outside of the Alvarium Go SDK
	Signature   string              `json:"signature,omitempty"` // Signature contains the signature of the party making the annotation
	IsSatisfied bool                `json:"isSatisfied"`         // IsSatisfied indicates whether the criteria defining the annotation were fulfilled
	Timestamp   time.Time           `json:"timestamp,omitempty"` // Timestamp indicates when the annotation was created
}

// NewAnnotation will map an Alvarium SDK annotation into an Annotation document
func NewAnnotation(a contracts.Annotation) Annotation {
	return Annotation{
		Key:         a.Id.String(),
		DataRef:     a.Key,
		Hash:        a.Hash,
		Host:        a.Host,
		Tag:         a.Tag,
		Layer:       a.Layer,
		Kind:        string(a.Kind),
		Signature:   a.Signature,
		IsSatisfied: a.IsSatisfied,
		Timestamp:   a.Timestamp,
	}
}

// Score represents a document in the "score" vertex collection
type Score struct {
	Key        ulid.ULID           `json:"_key,omitempty"`       // Key uniquely identifies the document in the database
	DataRef    string              `json:"dataRef,omitempty"`    // DataRef points to the key of the data being annotated
	Passed     int                 `json:"score,omitempty"`      // Passed indicates how many of the annotations for a given dataRef were Satisfied
	Count      int                 `json:"count,omitempty"`      // Count indicates the total number of annotations applicable to a dataRef
	Policy     string              `json:"policy,omitempty"`     // Policy will indicate some version of the policy used to calculate confidence
	Confidence float64             `json:"confidence,omitempty"` // Confidence is the percentage of trust in the dataRef
	Timestamp  time.Time           `json:"timestamp,omitempty"`  // Timestamp indicates when the score was calculated
	Tag        []string            `json:"tag,omitempty"`
	Layer      contracts.LayerType `json:"layer,omitempty"`
}

func uniqueTags(annotations []Annotation) []string {
	unique := make(map[string]bool)
	var result []string

	for _, annotation := range annotations {
		if !unique[annotation.Tag] {
			unique[annotation.Tag] = true
			result = append(result, annotation.Tag)
		}
	}
	return result
}

func NewScore(dataRef string, annotations []Annotation, policy policies.DcfPolicy, tagScores map[string]Score) Score {
	// All incoming annotations will have the same layer value
	layer := annotations[0].Layer

	// The received annotations might have multiple tag values
	// The score tag should contain all these tag values
	scoreTag := uniqueTags(annotations)

	var totalTagConfidence float64
	var totalWeight, passedWeight float32
	var passed int
	for _, a := range annotations {
		w := policy.FetchWeight(a.Kind)
		totalWeight += float32(w.Value)
		if a.IsSatisfied {
			passed++
			passedWeight += float32(w.Value)
		}

		tagScore, exists := tagScores[a.Tag]
		if exists {
			totalTagConfidence += tagScore.Confidence
		} else {
			// Default value that penalizes the score for not having stack confidence
			totalTagConfidence += 0.7
		}
	}

	averageTagConfidence := totalTagConfidence / float64(len(annotations))
	confidence := float64(passedWeight / totalWeight)
	confidence *= averageTagConfidence
	confidence = math.Round(confidence*100) / 100

	s := Score{
		Key:        NewULID(),
		DataRef:    dataRef,
		Passed:     passed,
		Count:      len(annotations),
		Policy:     policy.Name,
		Confidence: confidence,
		Timestamp:  time.Now(),
		Layer:      layer,
		Tag:        scoreTag,
	}
	return s
}

// Trust represents a document in the "trust" edge collection
type Trust struct {
	From string `json:"_from"`
	To   string `json:"_to"`
}

// Lineage represents a document in the "lineage" edge collection
type Lineage struct {
	From string `json:"_from"`
	To   string `json:"_to"`
}

// Scoring represents a document in the "scoring" edge collection
type Scoring struct {
	From string `json:"_from"`
	To   string `json:"_to"`
}

// Scoring represents a document in the "stack" edge collection
type Stack struct {
	From string `json:"_from"`
	To   string `json:"_to"`
}
