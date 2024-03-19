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

package populator_api

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/project-alvarium/alvarium-sdk-go/pkg/interfaces"
	"github.com/project-alvarium/scoring-apps-go/internal/db"
	"github.com/project-alvarium/scoring-apps-go/internal/hashprovider"
	"github.com/project-alvarium/scoring-apps-go/internal/models"
	"github.com/project-alvarium/scoring-apps-go/pkg/responses"
)

const (
	headerCORS           string = "Access-Control-Allow-Origin"
	headerCORSValue      string = "*"
	headerKeyContentType string = "Content-Type"
	headerValueJson      string = "application/json"
)

func LoadRestRoutes(r *mux.Router, dbArango *db.ArangoClient, dbMongo *db.MongoProvider, logger interfaces.Logger) {
	r.HandleFunc("/",
		func(w http.ResponseWriter, r *http.Request) {
			getIndexHandler(w, r, logger)
		}).Methods(http.MethodGet)

	r.HandleFunc("/data/{limit:[0-9]+}",
		func(w http.ResponseWriter, r *http.Request) {
			getSampleDataHandler(w, r, dbMongo, logger)
		}).Methods(http.MethodGet)

	r.HandleFunc("/data/count",
		func(w http.ResponseWriter, r *http.Request) {
			getDocumentCountHandler(w, r, dbMongo, logger)
		}).Methods(http.MethodGet)

	r.HandleFunc("/data/{id}/annotations",
		func(w http.ResponseWriter, r *http.Request) {
			getAnnotationsHandler(w, r, dbMongo, dbArango, logger)
		}).Methods(http.MethodGet)
}

func getIndexHandler(w http.ResponseWriter, r *http.Request, logger interfaces.Logger) {
	defer r.Body.Close()
	start := time.Now()
	w.Header().Add(headerCORS, headerCORSValue)
	w.Header().Add(headerKeyContentType, "text/html")
	w.Write([]byte("<html><head><title>Populator API</title></head><body><h2>Populator API</h2></body></html>"))

	elapsed := time.Now().Sub(start)
	logger.Write(slog.LevelDebug, fmt.Sprintf("getIndexHandler OK %v", elapsed))
}

func getDocumentCountHandler(w http.ResponseWriter, r *http.Request, dbMongo *db.MongoProvider, logger interfaces.Logger) {
	defer r.Body.Close()

	count, err := dbMongo.CountDocuments(r.Context())
	if err != nil {
		logger.Error(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	result := responses.DocumentCountResponse{Count: count}
	b, _ := json.Marshal(result)
	logger.Write(slog.LevelDebug, fmt.Sprintf("count=%v, %s", count, string(b)))
	w.Header().Add(headerKeyContentType, headerValueJson)
	w.Header().Add(headerCORS, headerCORSValue)
	w.WriteHeader(http.StatusOK)
	w.Write(b)
}

func getSampleDataHandler(w http.ResponseWriter, r *http.Request, dbMongo *db.MongoProvider, logger interfaces.Logger) {
	defer r.Body.Close()

	vars := mux.Vars(r)
	limit, err := strconv.Atoi(vars["limit"])
	if err != nil {
		logger.Write(slog.LevelDebug, "Bad request: "+err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}
	results, err := dbMongo.QueryMostRecent(r.Context(), limit)
	if err != nil {
		logger.Error(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	var viewModels []responses.DataViewModel
	for _, i := range results {
		viewModels = append(viewModels, models.ViewModelFromMongoRecord(i))
	}

	response := responses.DataListResponse{
		Count:     len(viewModels),
		Documents: viewModels,
	}

	b, _ := json.Marshal(response)
	w.Header().Add(headerKeyContentType, headerValueJson)
	w.Header().Add(headerCORS, headerCORSValue)
	w.WriteHeader(http.StatusOK)
	w.Write(b)
}

func getAnnotationsHandler(w http.ResponseWriter, r *http.Request, dbMongo *db.MongoProvider, dbArango *db.ArangoClient, logger interfaces.Logger) {
	defer r.Body.Close()

	vars := mux.Vars(r)
	id := vars["id"]
	if len(id) == 0 {
		errMsg := "Bad request: no id provided"
		logger.Write(slog.LevelDebug, errMsg)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(errMsg))
		return
	}

	record, err := dbMongo.FetchById(r.Context(), id)
	if err != nil {
		logger.Error(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	sampleData := models.SampleFromMongoRecord(record)
	b, _ := json.Marshal(sampleData)
	key := hashprovider.DeriveHash(b)

	annotations, err := dbArango.QueryAnnotations(r.Context(), key)
	if err != nil {
		logger.Error(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	response := responses.AnnotationListResponse{
		Count:       len(annotations),
		Annotations: annotations,
	}
	b, _ = json.Marshal(response)
	w.Header().Add(headerKeyContentType, headerValueJson)
	w.Header().Add(headerCORS, headerCORSValue)
	w.WriteHeader(http.StatusOK)
	w.Write(b)
}
