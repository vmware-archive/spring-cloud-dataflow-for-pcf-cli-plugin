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
package download

import (
	"strings"
	"os"
	"fmt"
	"net/http"
	"io"
	"path"
)

func DownloadFile(url string) (string, error) {
	tokens := strings.Split(url, "/")
	fileName := tokens[len(tokens)-1]

	targetDir, err := targetDirectory()
	if err != nil {
		return "", err
	}
	filePath := path.Join(targetDir, fileName)

	_, err = os.Stat(filePath) // FIXME: proper caching using e.g. etags or SHAs to be implemented
	if err != nil {
		fmt.Printf("Downloading %s\n", url)
		file, err := os.Create(filePath)
		if err != nil {
			fmt.Printf("Error creating %s: %s\n", filePath, err)
			return "", err
		}
		defer file.Close()

		response, err := http.Get(url)
		if err != nil {
			fmt.Printf("Error accessing %s: %s\n", url, err)
			return "", err
		}
		defer response.Body.Close()

		_, err = io.Copy(file, response.Body)
		if err != nil {
			fmt.Printf("Error downloading %s: %s\n", url, err)
			return "", err
		}
	}

	return filePath, nil
}

func targetDirectory() (string, error) {
	dir := os.Getenv("CF_HOME")
	if dir == "" {
		dir = os.Getenv("HOME")
	}
	dirPath := path.Join(dir, ".cf", "spring-cloud-dataflow-for-pcf/cache")
	return dirPath, os.MkdirAll(dirPath, 0755)
}
