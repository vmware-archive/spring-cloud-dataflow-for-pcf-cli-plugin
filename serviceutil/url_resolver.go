/*
 * Copyright (C) 2017-Present Pivotal Software, Inc. All rights reserved.
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
package serviceutil

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"errors"

	"code.cloudfoundry.org/cli/plugin"
	"github.com/pivotal-cf/spring-cloud-dataflow-for-pcf-cli-plugin/httpclient"
)

type serviceDefinitionResp struct {
	Credentials struct {
		URI string
	}
}

// ServiceInstanceURL obtains the service instance URL of a service with a specific name. This is a secure operation and an access token is provided for authentication and authorisation.
func ServiceInstanceURL(cliConnection plugin.CliConnection, serviceInstanceName string, accessToken string, authClient httpclient.AuthenticatedClient) (string, error) {
	serviceModel, err := cliConnection.GetService(serviceInstanceName)
	if err != nil {
		return "", fmt.Errorf("Service instance not found: %s", err)
	}

	parsedUrl, err := url.Parse(serviceModel.DashboardUrl)
	if err != nil {
		return "", err
	}
	path := parsedUrl.Path

	segments := strings.Split(path, "/")
	if len(segments) == 0 || (len(segments) == 1 && segments[0] == "") {
		return "", fmt.Errorf("path of %s has no segments", serviceModel.DashboardUrl)
	}

	parsedUrl.Path = strings.Join(segments[:len(segments)-1], "/")

	_, statusCode, header, err := authClient.DoAuthenticatedGet(parsedUrl.String(), accessToken)
	if statusCode != http.StatusFound {
		if err != nil {
			return "", fmt.Errorf("service broker failed: %s", err)
		}
		return "", fmt.Errorf("service broker did not return expected response (302): %d", statusCode)
	}

	locationHeader, locationPresent := header["Location"]
	if !locationPresent {
		return "", errors.New("service broker did not return a location header")
	}

	if len(locationHeader) != 1 {
		return "", fmt.Errorf("service broker returned a location header of the wrong length (%d)", len(locationHeader))
	}

	return locationHeader[0], nil
}
