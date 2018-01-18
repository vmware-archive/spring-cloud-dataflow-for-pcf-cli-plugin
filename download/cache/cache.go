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
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"hash"
)

const (
	cacheEntriesFileName = ".cachedata"
	cfHomeProperty       = "CF_HOME"
	homeProperty         = "HOME"
	cfDataDirectory      = ".cf"
	cacheEntriesFilePerm = 0644
	cacheDirectoryPerm   = 0755
)

var scdfCacheDirectory = path.Join("spring-cloud-dataflow-for-pcf", "cache")

// Cache provides a cache of files indexed by their download URLs. Each cached file has an associated etag.
//go:generate counterfeiter -o ../downloadfakes/fake_cache.go . Cache
type Cache interface {
	Entry(Url string) CacheEntry
}

type fileCache struct {
	downloadsDirectory string
	etagHelper         EtagHelper
	progressWriter     io.Writer
}

func (f *fileCache) Entry(Url string) CacheEntry {
	return &fileCacheEntry{
		downloadUrl:        Url,
		downloadFile:       createFilePathForDownloadFile(Url, f.downloadsDirectory),
		checksumCalculator: &checksumCalculator{},
		etagHelper:         f.etagHelper,
		progressWriter:     f.progressWriter,
	}
}

func NewCache(progressWriter io.Writer) (*fileCache, error) {
	downloadsDir, err := getDownloadsDirectory()
	if err != nil {
		return nil, err
	}

	cacheDataFile := path.Join(downloadsDir, cacheEntriesFileName)
	etagHelper, err := NewEtagIndex(cacheDataFile)
	if err != nil {
		return nil, err
	}

	return &fileCache{
		downloadsDirectory: downloadsDir,
		etagHelper:         etagHelper,
		progressWriter:     progressWriter,
	}, nil
}

// Place checksum calculator functionality inside an interface to help with testing
//go:generate counterfeiter -o ../downloadfakes/fake_checksumcalculator.go . ChecksumCalculator
type ChecksumCalculator interface {
	CalculateChecksum(filePath string, hashFunc hash.Hash) (string, error)
}

type checksumCalculator struct {
}

func (c *checksumCalculator) CalculateChecksum(filePath string, hash hash.Hash) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	if _, err := io.Copy(hash, f); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// Place etag handling functionality inside an interface to help with testing
//go:generate counterfeiter -o ../downloadfakes/fake_etaghelper.go . EtagHelper
type EtagHelper interface {
	GetETagForUrl(url string) (string, error)
	SetEtagForUrl(url string, etag string) error
}

// CacheEntry provides a cache of a single file and its etag.
//go:generate counterfeiter -o ../downloadfakes/fake_cacheentry.go . CacheEntry
type CacheEntry interface {
	// Retrieve returns the fully qualified path of the cached file and its etag.  If the file has not been cached, the returned path is empty.
	Retrieve() (path string, etag string, err error)

	// Store writes the cached file contents and associates the given etag (which  may be empty) with the file.
	// If the file contents cannot be written or the etag associated with the file, an error is returned.
	// The file contents are checked against the given checksum using the given hash and an error is returned if the check fails.
	Store(contents io.ReadCloser, etag string, checksum string, hashFunc hash.Hash) error
}

type fileCacheEntry struct {
	downloadUrl        string
	downloadFile       string
	checksumCalculator ChecksumCalculator
	etagHelper         EtagHelper
	progressWriter     io.Writer
}

func (f *fileCacheEntry) Retrieve() (path string, etag string, err error) {
	if fileExists(f.downloadFile) {
		path = f.downloadFile
	} else {
		path = ""
	}

	etag, err = f.etagHelper.GetETagForUrl(f.downloadUrl)
	return path, etag, err
}

func (f *fileCacheEntry) Store(contents io.ReadCloser, etag string, checksum string, hash hash.Hash) error {
	err := writeDataToNamedFile(contents, f.downloadFile)
	if err != nil {
		fmt.Fprintf(f.progressWriter, "Error downloading %s: %s\n", f.downloadFile, err)
		return err
	}

	calculatedCheckSum, err := f.checksumCalculator.CalculateChecksum(f.downloadFile, hash)
	if err != nil {
		fmt.Fprintf(f.progressWriter, "Error calculating checksum of %s: %s\n", f.downloadFile, err)
		return err
	}

	if checksum != calculatedCheckSum {
		return fmt.Errorf("Downloaded file '%s' checksum does not match supplied value '%s'", f.downloadFile, checksum)
	}

	if etag != "" {
		err = f.etagHelper.SetEtagForUrl(f.downloadUrl, etag)
		if err != nil {
			return err
		}
	}

	return nil
}

func writeDataToNamedFile(data io.ReadCloser, filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}

	defer file.Close()
	defer data.Close()

	_, err = io.Copy(file, data)
	if err != nil {
		return err
	}

	return nil
}

func fileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return !os.IsNotExist(err)
}

func createDownloadsDirectory(dirPath string) error {
	return os.MkdirAll(dirPath, cacheDirectoryPerm)
}

func createFilePathForDownloadFile(url string, destinationDirectory string) string {
	tokens := strings.Split(url, string(os.PathSeparator))
	fileName := tokens[len(tokens)-1]
	filePath := path.Join(destinationDirectory, fileName)
	return filePath
}

func getDownloadsDirectory() (string, error) {
	dir := os.Getenv(cfHomeProperty)
	if dir == "" {
		dir = os.Getenv(homeProperty)
	}
	dirPath := path.Join(dir, cfDataDirectory, scdfCacheDirectory)
	return dirPath, createDownloadsDirectory(dirPath)
}
