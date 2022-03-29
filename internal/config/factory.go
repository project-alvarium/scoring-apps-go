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
	"errors"
	"path"
	"strings"
)

// NewReader returns a type that will hydrate an ApplicationConfig instance from a file.
// Currently only "json" is supported as a value for the readerType parameter. Intention
// is to extend to TOML at some point.
func NewReader(readerType string) (Reader, error) {
	var reader Reader
	if readerType == "json" {
		reader = newJsonReader()
	} else {
		return reader, errors.New("Unsupported readerType value: " + readerType)
	}
	return reader, nil
}

func GetFileExtension(cfgPath string) string {
	tokens := strings.Split(path.Base(cfgPath), ".")
	if len(tokens) == 2 {
		return tokens[1]
	}
	return tokens[0]
}
