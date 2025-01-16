package fileutil

import (
	"os"
	"sync"
)

type fileCache struct {
	filepath string
	bytes    []byte
}

var (
	fileCacheStore map[string]*fileCache
	rwMutex        sync.RWMutex
)

func init() {
	fileCacheStore = make(map[string]*fileCache)
}

func ReadFileWithCache(filepath string) ([]byte, error) {
	rwMutex.RLock()
	defer rwMutex.RUnlock()

	if cache, exists := fileCacheStore[filepath]; exists {
		go func() {
			bytes, err := os.ReadFile(filepath)
			if err != nil {
				return
			}
			rwMutex.Lock()
			defer rwMutex.Unlock()
			fileCacheStore[filepath] = &fileCache{filepath, bytes}
		}()
		return cache.bytes, nil
	}

	bytes, err := os.ReadFile(filepath)
	if err != nil {
		return nil, err
	}

	rwMutex.Lock()
	defer rwMutex.Unlock()
	fileCacheStore[filepath] = &fileCache{filepath, bytes}

	return bytes, nil
}
