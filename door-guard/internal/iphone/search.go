package iphone

import (
	"net"
	"time"
)

func isHostAvailable(hostName string) bool {
	_, err := net.LookupHost(hostName)
	return err == nil
}

func IsIphoneNearby(hostName string) (bool, error) {
	maxRetries := 5
	retryInterval := 2 * time.Second // リトライ間隔

	for i := 0; i <= maxRetries; i++ {
		if isHostAvailable(hostName) {
			return true, nil
		}
		time.Sleep(retryInterval)
	}

	return false, nil
}
