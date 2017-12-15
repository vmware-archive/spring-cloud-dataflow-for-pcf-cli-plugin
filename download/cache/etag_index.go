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
	"bufio"
	"io/ioutil"
	"os"
	"strings"
)

type etagIndex struct {
	indexFile string
}

func NewEtagIndex(indexFile string) (*etagIndex, error) {
	if !fileExists(indexFile) {
		f, err := os.OpenFile(indexFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, cacheEntriesFilePerm)
		if err != nil {
			return nil, err
		}
		f.Close()
	}

	return &etagIndex{
		indexFile: indexFile,
	}, nil
}

func (h *etagIndex) GetETagForUrl(url string) (string, error) {
	cacheDataFile, err := os.Open(h.indexFile)
	if err != nil {
		return "", err
	}

	defer cacheDataFile.Close()

	scanner := bufio.NewScanner(cacheDataFile)
	for scanner.Scan() {
		scannedLine := scanner.Text()
		if strings.HasPrefix(scannedLine, url+" : ") {
			return strings.Split(scannedLine, " : ")[1], nil
		}
	}

	return "", nil
}

func (h *etagIndex) SetEtagForUrl(url string, etag string) error {
	cacheBytes, err := ioutil.ReadFile(h.indexFile)
	if err != nil {
		return err
	}

	cacheLines := strings.Split(string(cacheBytes), "\n")

	existingEntryFound := false
	for i, line := range cacheLines {
		if strings.HasPrefix(line, url+" : ") {
			cacheLines[i] = url + " : " + etag
			existingEntryFound = true
			break
		}
	}

	if !existingEntryFound {
		cacheLines = append(cacheLines, url+" : "+etag)
	}

	output := strings.Join(cacheLines, "\n")
	return ioutil.WriteFile(h.indexFile, []byte(output), cacheEntriesFilePerm)
}
