package catbox

import (
	"bufio"
	"fmt"
	"os"
	"sync"
)

type ProxyManager struct {
	proxies []string
	index   int
	mu      sync.Mutex
}

func InitProxyManager(proxyFile string) (*ProxyManager, error) {
	file, err := os.Open(proxyFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var proxies []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		proxy := scanner.Text()
		if proxy != "" {
			proxies = append(proxies, proxy)
		}
	}
	if len(proxies) == 0 {
		return nil, fmt.Errorf("no proxies found in %s", proxyFile)
	}
	return &ProxyManager{proxies: proxies}, nil
}

func (pm *ProxyManager) GetNextProxy() string {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	proxy := pm.proxies[pm.index]
	pm.index = (pm.index + 1) % len(pm.proxies)
	return proxy
}
