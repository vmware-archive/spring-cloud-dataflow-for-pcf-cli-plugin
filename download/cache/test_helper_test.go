package cache

type FieldGetter interface {
	GetDownloadUrl() string
	GetDownloadFile() string
	GetCacheEntriesFile() string
}

func (f *fileCacheEntry) GetDownloadUrl() string {
	return f.downloadUrl
}

func (f *fileCacheEntry) GetDownloadFile() string {
	return f.downloadFile
}

func (f *fileCacheEntry) GetCacheEntriesFile() string {
	return f.cacheEntriesFile
}

type FieldSetter interface {
	SetChecksumCalculator(calculator ChecksumCalculator)
	SetEtagHelper(helper EtagHelper)
	SetDownloadsFile(filePath string)
}

func (f *fileCacheEntry) SetChecksumCalculator(calculator ChecksumCalculator) {
	f.checksumCalculator = calculator
}

func (f *fileCacheEntry) SetEtagHelper(helper EtagHelper) {
	f.etagHelper = helper
}

func (f *fileCacheEntry) SetDownloadsFile(filePath string) {
	f.downloadFile = filePath
}
