/*
 * Copyright (C) 2017-Present Pivotal Software, Inc. All rights reserved.
 *
 * This program and the accompanying materials are made available under
 * the terms of the under the Apache License, Version 2.0 (the "License”);
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package cache

import (
	"encoding/json"
	"io/ioutil"
)

type etagIndex struct {
	indexFile string
}

type IndexMap map[string]string

func NewEtagIndex(indexFile string) (*etagIndex, error) {
	h := &etagIndex{
		indexFile: indexFile,
	}

	if !fileExists(indexFile) {
		err := h.writeIndex(IndexMap{})
		if err != nil {
			return nil, err
		}
	}

	return h, nil
}

func (h *etagIndex) GetETagForUrl(url string) (string, error) {
	index := IndexMap{}
	err := h.readIndex(index)
	if err != nil {
		return "", err
	}
	return index[url], nil
}

func (h *etagIndex) SetEtagForUrl(url string, etag string) error {
	index := IndexMap{}
	err := h.readIndex(index)
	if err != nil {
		return err
	}

	index[url] = etag

	return h.writeIndex(index)
}

func (h *etagIndex) writeIndex(index IndexMap) error {
	bytes, err := json.Marshal(index)
	if err != nil {
		return err // Should never get here
	}
	return ioutil.WriteFile(h.indexFile, bytes, cacheEntriesFilePerm)
}

func (h *etagIndex) readIndex(index IndexMap) error {
	bytes, err := ioutil.ReadFile(h.indexFile)
	if err != nil {
		return err
	}

	return json.Unmarshal(bytes, &index)
}
