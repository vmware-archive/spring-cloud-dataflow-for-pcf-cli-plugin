package cache

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"crypto/sha256"

	"io/ioutil"
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
	cacheEntriesFile   string
}

func (f *fileCache) Entry(Url string) CacheEntry {
	return &fileCacheEntry{
		downloadUrl:      Url,
		downloadFile:     createFilePathForDownloadFile(Url, f.downloadsDirectory),
		cacheEntriesFile: f.cacheEntriesFile,
	}
}

func NewCache() (*fileCache, error) {
	downloadsDir, err := getDownloadsDirectory()
	if err != nil {
		return nil, err
	}

	cacheDataFile := path.Join(downloadsDir, cacheEntriesFileName)
	if !fileExists(cacheDataFile) {
		_, err := os.Create(cacheDataFile)
		if err != nil {
			return nil, err
		}

		err = os.Chmod(cacheDataFile, cacheEntriesFilePerm)
		if err != nil {
			return nil, err
		}
	}

	return &fileCache{
		downloadsDirectory: downloadsDir,
		cacheEntriesFile:   cacheDataFile,
	}, nil
}

// CacheEntry provides a cache of a single file and its etag.
//go:generate counterfeiter -o ../downloadfakes/fake_cacheentry.go . CacheEntry
type CacheEntry interface {
	// Retrieve returns the fully qualified path of the cached file and its etag.  If the file has not been cached, the returned path is empty.
	Retrieve() (path string, etag string, err error)

	// Store writes the cached file contents and associates the given etag (which  may be empty) with the file.
	// If the file contents cannot be written or the etag associated with the file, an error is returned.
	// The file contents are checked against the given checksum and an error is returned if the check fails.
	Store(contents io.ReadCloser, etag string, checksum string) error
}

type fileCacheEntry struct {
	downloadUrl      string
	downloadFile     string
	cacheEntriesFile string
}

func (f *fileCacheEntry) Retrieve() (path string, etag string, err error) {
	if fileExists(f.downloadFile) {
		path = f.downloadFile
	} else {
		path = ""
	}

	etag, err = getETagForUrl(f.downloadUrl, f.cacheEntriesFile)
	return path, etag, err
}

func (f *fileCacheEntry) Store(contents io.ReadCloser, etag string, checksum string) error {
	err := writeDataToNamedFile(contents, f.downloadFile)
	if err != nil {
		fmt.Printf("Error downloading %s: %s\n", f.downloadFile, err)
		return err
	}

	calculatedCheckSum, err := calculateChecksumOfFile(f.downloadFile)
	if err != nil {
		fmt.Printf("Error calculating checksum of %s: %s\n", f.downloadFile, err)
		return err
	}

	if checksum != calculatedCheckSum {
		return errors.New(fmt.Sprintf("Downloaded file '%s' checksum does not match supplied value", f.downloadFile))
	}

	if etag != "" {
		err = setEtagForUrl(f.downloadUrl, etag, f.cacheEntriesFile)
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

func calculateChecksumOfFile(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}

	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func fileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return !os.IsNotExist(err)
}

func createDownloadsDirectory(dirPath string) error {
	return os.MkdirAll(dirPath, cacheDirectoryPerm)
}

func createFilePathForDownloadFile(url string, destinationDirectory string) string {
	tokens := strings.Split(url, "/")
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

func getETagForUrl(url string, metadataFile string) (string, error) {
	cacheDataFile, err := os.Open(metadataFile)
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

func setEtagForUrl(url string, etag string, cacheEntriesFile string) error {
	cacheBytes, err := ioutil.ReadFile(cacheEntriesFile)
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
	return ioutil.WriteFile(cacheEntriesFile, []byte(output), cacheEntriesFilePerm)
}
