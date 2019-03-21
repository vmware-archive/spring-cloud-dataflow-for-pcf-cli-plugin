/*
 * Copyright (C) 2018-Present Pivotal Software, Inc. All rights reserved.
 *
 * This program and the accompanying materials are made available under
 * the terms of the under the Apache License, Version 2.0 (the "License”);
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package skipper

import (
	"crypto/sha256"
	"hash"

	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

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

func SkipperShellDownloadUrl(skipperServer string, authClient httpclient.AuthenticatedClient, accessToken string) (string, string, hash.Hash, error) {
	defaultHashFunc := sha256.New()
	bodyReader, statusCode, _, err := authClient.DoAuthenticatedGet(skipperServer+"/about", accessToken)
	if err != nil {
		return "", "", defaultHashFunc, fmt.Errorf("Skipper server error: %s", err)
	}
	if statusCode != http.StatusOK {
		return "", "", defaultHashFunc, fmt.Errorf("Skipper server failed: %d", statusCode)
	}
	body, err := ioutil.ReadAll(bodyReader)
	if err != nil {
		return "", "", defaultHashFunc, fmt.Errorf("Cannot read Skipper server response body: %s", err)
	}

	var aboutResp AboutResp
	err = json.Unmarshal(body, &aboutResp)
	if err != nil {
		return "", "", defaultHashFunc, fmt.Errorf("Invalid Skipper server response JSON: %s, response body: '%s'", err, string(body))
	}

	shellInfo := aboutResp.VersionInfo.Shell

	if shellInfo.ChecksumSha256 != "" {
		return shellInfo.Url, shellInfo.ChecksumSha256, defaultHashFunc, nil
	}

	return shellInfo.Url, shellInfo.ChecksumSha1, sha1.New(), nil
}
