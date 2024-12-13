package catbox

import (
	"database/sql"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"time"
)

func Worker(db *sql.DB, idChan <-chan string, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()

	for id := range idChan {
		if G_state != StateRunning {
			handleNonRunningState()
			continue
		}

		processID(db, id)
	}
}

func handleNonRunningState() {
	switch G_state {
	case StatePaused:
		time.Sleep(time.Second)
	case StateStopped:
		return
	}
}

func processID(db *sql.DB, id string) {
	for _, ext := range G_config.AllowedExt {
		if !retryUntilSuccess(func() (bool, error) { return checkFileExists(id, ext) }, G_config.TryLimit) {
			G_logger.Debugf("File does not exist: %s%s", id, ext)
			continue
		}

		G_Found_Per_Min.Add(1)
		url := fmt.Sprintf("%s%s%s", G_config.BaseURL, id, ext)
		if err := insertFileRecord(db, id, url, ext); err != nil {
			G_logger.Debugf("Failed to insert file record: %v", err)
		}

		if G_config.WebhookEnabled {
			sendToWebhook(url)
		}

		if G_config.DownloadEnabled {
			downloadFileWithRetry(id, ext)
		}
	}
}

func retryUntilSuccess(action func() (bool, error), maxRetries int) bool {
	for tries := maxRetries; tries > 0; tries-- {
		success, err := action()
		if err != nil {
			G_logger.Debugf("Error encountered, retries left: %d, error: %v", tries-1, err)
			continue
		}
		if success {
			return true
		}
	}
	return false
}

func insertFileRecord(db *sql.DB, id, url, ext string) error {
	_, err := db.Exec("INSERT INTO valid_ids (id, url, ext) VALUES (?, ?, ?)", id, url, ext)
	return err
}

func sendToWebhook(url string) {
	if err := SendMessageToWebhook(url); err != nil {
		G_logger.Errorf("Failed to send to webhook: %v", err)
	}
}

func downloadFileWithRetry(id, ext string) {
	retryUntilSuccess(func() (bool, error) {
		if err := downloadFile(id, ext); err != nil {
			return false, err
		}
		return true, nil
	}, G_config.TryLimit)
}

func checkFileExists(id, ext string) (bool, error) {
	url := fmt.Sprintf("%s%s%s", G_config.BaseURL, id, ext)
	client := createHttpClient()

	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return false, err
	}

	resp, err := client.Do(req)
	if err != nil {
		G_logger.Debugf("Request error for %s: %v", url, err)
		return false, nil
	}
	defer resp.Body.Close()

	G_Req_Per_Sec.Add(1)
	return resp.StatusCode == http.StatusOK, nil
}

func createHttpClient() *http.Client {
	if G_config.UseProxies {
		proxy := G_proxyManager.GetNextProxy()
		G_logger.Debugf("Using proxy: %s", proxy)
		proxyURL, _ := url.Parse(fmt.Sprintf("%s://%s", G_config.ProxyType, proxy))
		return &http.Client{
			Transport: &http.Transport{
				Proxy:               http.ProxyURL(proxyURL),
				DialContext:         (&net.Dialer{Timeout: 5 * time.Second}).DialContext,
				TLSHandshakeTimeout: 5 * time.Second,
			},
			Timeout: 5 * time.Second,
		}
	}
	return &http.Client{Timeout: 2 * time.Second}
}

func downloadFile(id, ext string) error {
	url := fmt.Sprintf("%s%s%s", G_config.BaseURL, id, ext)
	filePath := filepath.Join(G_config.DownloadPath, id+ext)

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed, status: %d", resp.StatusCode)
	}

	out, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", filePath, err)
	}
	defer out.Close()

	if _, err = io.Copy(out, resp.Body); err != nil {
		return fmt.Errorf("failed to write file %s: %w", filePath, err)
	}

	G_logger.Debugf("Downloaded: %s", url)
	return nil
}
