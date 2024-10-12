package catbox

import (
	"os"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Workers         int      `yaml:"workers"`
	RetryLimit      int      `yaml:"retry_limit"`
	DownloadEnabled bool     `yaml:"download_enabled"`
	DownloadPath    string   `yaml:"download_path"`
	AllowedExt      []string `yaml:"allowed_extensions"`
	LogLevel        string   `yaml:"log_level"`
	Database        string   `yaml:"database"`
	BaseURL         string   `yaml:"base_url"`
	ProxyFile       string   `yaml:"proxy_file"`
	ProxyType       string   `yaml:"proxy_type"`
	UseProxies      bool     `yaml:"use_proxies"`
	WebhookURL      string   `yaml:"webhook_url"`
	WebhookEnabled  bool     `yaml:"webhook_enabled"`
}

func LoadConfig() error {
	file, err := os.Open("config.yaml")
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&G_config); err != nil {
		return err
	}

	level, err := logrus.ParseLevel(G_config.LogLevel)
	if err != nil {
		return err
	}
	G_logger.SetLevel(level)

	return nil
}
