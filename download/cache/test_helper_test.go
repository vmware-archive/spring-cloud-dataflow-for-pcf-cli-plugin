package cache

type FieldGetter interface {
	GetDownloadUrl() string
	GetDownloadFile() string
	GetCacheEntriesFile() string
}

func (ce *fileCacheEntry) GetDownloadUrl() string {
	return ce.downloadUrl
}

func (ce *fileCacheEntry) GetDownloadFile() string {
	return ce.downloadFile
}

func (ce *fileCacheEntry) GetCacheEntriesFile() string {
	return ce.cacheEntriesFile
}
