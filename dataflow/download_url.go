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
package dataflow

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"crypto/sha256"
	"hash"

	"crypto/sha1"

	"github.com/pivotal-cf/spring-cloud-dataflow-for-pcf-cli-plugin/httpclient"
)

type AboutResp struct {
	VersionInfo struct {
		Shell struct {
			Url            string
			ChecksumSha1   string
			ChecksumSha256 string
		}
	}
}

func DataflowShellDownloadUrl(dataflowServer string, authClient httpclient.AuthenticatedClient, accessToken string) (string, string, hash.Hash, error) {
	defaultHashFunc := sha256.New()
	bodyReader, statusCode, _, err := authClient.DoAuthenticatedGet(dataflowServer+"/about", accessToken)
	if err != nil {
		return "", "", defaultHashFunc, fmt.Errorf("Dataflow server error: %s", err)
	}
	if statusCode != http.StatusOK {
		return "", "", defaultHashFunc, fmt.Errorf("Dataflow server failed: %d", statusCode)
	}
	body, err := ioutil.ReadAll(bodyReader)
	if err != nil {
		return "", "", defaultHashFunc, fmt.Errorf("Cannot read dataflow server response body: %s", err)
	}

	var aboutResp AboutResp
	err = json.Unmarshal(body, &aboutResp)
	if err != nil {
		return "", "", defaultHashFunc, fmt.Errorf("Invalid dataflow server response JSON: %s, response body: '%s'", err, string(body))
	}

	shellInfo := aboutResp.VersionInfo.Shell

	// FIXME: delete this temporary code
	if shellInfo.Url == "" {
		shellInfo.Url = "https://repo.spring.io/libs-snapshot/org/springframework/cloud/spring-cloud-dataflow-shell/1.2.3.RELEASE/spring-cloud-dataflow-shell-1.2.3.RELEASE.jar"
		shellInfo.ChecksumSha256 = "9dec3eab5740cb087d7842bcb6bf924f9e008638dedeca16c5336bbc3c0e4453"
	}

	if shellInfo.ChecksumSha256 != "" {
		return shellInfo.Url, shellInfo.ChecksumSha256, defaultHashFunc, nil
	}

	return shellInfo.Url, shellInfo.ChecksumSha1, sha1.New(), nil
}
