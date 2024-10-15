package catbox

import (
	"context"
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

func Worker(ctx context.Context, db *sql.DB, idChan <-chan string, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			G_logger.Debug("Worker received stop signal. Exiting...")
			return
		default:
			switch G_state {
			case StateRunning:
				for id := range idChan {
					for _, ext := range G_config.AllowedExt {
						retries := G_config.RetryLimit
						var exists bool
						var err error

						for retries > 0 {
							exists, err = checkFileExists(id, ext)
							if err != nil {
								retries--
								if retries > 0 {
									G_logger.Debugf("Retrying (%d retries left)...", retries)
								} else {
									G_logger.Debugf("Max retries reached. Skipping %s.", id)
								}
							} else {
								break
							}
						}

						if exists {
							urll := fmt.Sprintf("%s%s%s", G_config.BaseURL, id, ext)
							_, err := db.Exec("INSERT INTO valid_ids (id, url, ext) VALUES (?, ?, ?)", id, urll, ext)
							if err != nil {
								G_logger.Errorf("Failed to insert file record into database: %v", err)
							}

							if G_config.WebhookEnabled {
								err = SendMessageToWebhook(urll)
								if err != nil {
									G_logger.Errorf("Failed to send message to webhook: %v", err)
								}
							}

							if G_config.DownloadEnabled {
								downloadRetries := G_config.RetryLimit
								for downloadRetries > 0 {
									err := downloadFile(id, ext)
									if err != nil {
										downloadRetries--
										if downloadRetries > 0 {
											G_logger.Debugf("Retrying download (%d retries left)...", downloadRetries)
										} else {
											G_logger.Errorf("Max retries reached for download. Skipping %s.%s.", id, ext)
										}
									} else {
										break
									}
								}
							}
						} else {
							G_logger.Debugf("File does not exist: %s%s", id, ext)
						}
					}
				}
			case StatePaused:
				time.Sleep(time.Second)
				continue
			case StateStopped:
				return
			}
		}
	}
}

func checkFileExists(id, ext string) (bool, error) {
	var client *http.Client
	url_s := fmt.Sprintf("%s%s%s", G_config.BaseURL, id, ext)

	if G_config.UseProxies {
		proxy := G_proxyManager.GetNextProxy()
		G_logger.Debugf("Using proxy: %s", proxy)

		proxyURL, err := url.Parse(fmt.Sprintf("%s://%s", G_config.ProxyType, proxy))
		if err != nil {
			return false, err
		}
		transport := &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
			DialContext: (&net.Dialer{
				Timeout: 5 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout: 7 * time.Second,
		}
		client = &http.Client{Transport: transport, Timeout: 10 * time.Second}
	} else {
		client = &http.Client{Timeout: 10 * time.Second}
	}

	req, err := http.NewRequest("HEAD", url_s, nil)
	if err != nil {
		return false, err
	}

	resp, err := client.Do(req)
	if err != nil {
		G_logger.Debugf("Error making request for %s: %v", url_s, err)
		return false, nil
	}
	defer resp.Body.Close()

	G_Req_Per_Sec.Add(1)

	if resp.StatusCode == http.StatusOK {
		return true, nil
	}
	return false, nil
}

func downloadFile(id, ext string) error {
	url := fmt.Sprintf("%s%s%s", G_config.BaseURL, id, ext)
	filePath := filepath.Join(G_config.DownloadPath, id+ext)

	resp, err := http.Get(url)
	if err != nil {
		G_logger.Errorf("Failed to download %s: %v", url, err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		G_logger.Errorf("Failed to download %s: Status %d", url, resp.StatusCode)
		return err
	}

	out, err := os.Create(filePath)
	if err != nil {
		G_logger.Errorf("Failed to create file %s: %v", filePath, err)
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		G_logger.Errorf("Failed to write file %s: %v", filePath, err)
		return err
	}

	G_logger.Debugf("Successfully downloaded: %s", url)
	return nil
}
