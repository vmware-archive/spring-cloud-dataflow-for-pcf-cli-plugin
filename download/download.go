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
	"errors"
	"fmt"
	"io"
	"net/http"

	"hash"

	"github.com/pivotal-cf/spring-cloud-dataflow-for-pcf-cli-plugin/download/cache"
)

const (
	ifNoneMatchHeader = "If-None-Match"
	etagHeader        = "ETag"
)

// Wrap Http response object actions inside an interface whose behaviour can be faked in tests
//go:generate counterfeiter -o downloadfakes/fake_httpresponse.go . HttpResponse
type HttpResponse interface {
	GetHeader(name string) string
	GetStatusCode() int
	GetBody() io.ReadCloser
}

type httpResponse struct {
	response *http.Response
}

func (h *httpResponse) GetHeader(name string) string {
	return h.response.Header.Get(name)
}

func (h *httpResponse) GetStatusCode() int {
	return h.response.StatusCode
}

func (h *httpResponse) GetBody() io.ReadCloser {
	return h.response.Body
}

func NewHttpResponse(response *http.Response) *httpResponse {
	return &httpResponse{
		response: response,
	}
}

// Added to allow testing of HTTP client usage
//go:generate counterfeiter -o downloadfakes/fake_httpclient.go . HttpClient
type HttpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type httpClient struct {
	client *http.Client
}

func (hc *httpClient) Do(req *http.Request) (*http.Response, error) {
	return hc.client.Do(req)
}

// Wrap Http request interactions inside an interface whose behaviour can be faked in tests
//go:generate counterfeiter -o downloadfakes/fake_httprequest.go . HttpRequest
type HttpRequest interface {
	SetHeader(key string, value string)
	SendRequest() (HttpResponse, error)
}

type httpRequest struct {
	client  HttpClient
	request *http.Request
}

func (h *httpRequest) SetHeader(key string, value string) {
	h.request.Header.Add(key, value)
}

func (h *httpRequest) SendRequest() (HttpResponse, error) {
	resp, err := h.client.Do(h.request)
	if err != nil {
		return nil, err
	}

	return NewHttpResponse(resp), nil
}

//go:generate counterfeiter -o downloadfakes/fake_httphelper.go . HttpHelper
type HttpHelper interface {
	CreateHttpRequest(method string, url string) (HttpRequest, error)
}

type httpHelper struct {
}

func (h *httpHelper) CreateHttpRequest(method string, url string) (HttpRequest, error) {
	cl := &http.Client{
		CheckRedirect: nil,
	}

	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}

	return &httpRequest{
		client:  &httpClient{cl},
		request: req,
	}, nil
}

func NewHttpHelper() *httpHelper {
	return &httpHelper{}
}

type Downloader interface {
	DownloadFile(url string, checksum string, hashFunc hash.Hash) (string, error)
}

type downloader struct {
	cache      cache.Cache
	httpHelper HttpHelper
}

func NewDownloader(cache cache.Cache, httpHelper HttpHelper) (*downloader, error) {
	return &downloader{
		cache:      cache,
		httpHelper: httpHelper,
	}, nil
}

func (d *downloader) DownloadFile(url string, checksum string, hashFunc hash.Hash) (string, error) {
	cacheEntry := d.cache.Entry(url)

	downloadedFilePath, cachedEtag, err := cacheEntry.Retrieve()
	if err != nil {
		return "", err
	}

	getRequest, err := d.httpHelper.CreateHttpRequest(http.MethodGet, url)
	if err != nil {
		return downloadedFilePath, fmt.Errorf("CreateHttpRequest for download URL %q failed: %s", url, err)
	}

	if cachedEtag != "" {
		getRequest.SetHeader(ifNoneMatchHeader, cachedEtag)

		if downloadedFilePath == "" {
			fmt.Printf("File at '%s' has previously been cached but cannot be found on local disk. Downloading again.\n", url)
		}
	}

	response, err := getRequest.SendRequest()
	if err != nil {
		return downloadedFilePath, fmt.Errorf("Download from URL %q failed: %s", url, err)
	}

	if response.GetStatusCode() == http.StatusNotModified {
		return downloadedFilePath, nil
	}

	if response.GetStatusCode() == http.StatusOK {
		fmt.Printf("Downloading %s\n", url)
		newEtagValue := response.GetHeader(etagHeader)
		err = cacheEntry.Store(response.GetBody(), newEtagValue, checksum, hashFunc)
		return downloadedFilePath, err
	}

	return downloadedFilePath, errors.New(fmt.Sprintf("Unexpected response '%d' downloading from '%s'", response.GetStatusCode(), url))
}
