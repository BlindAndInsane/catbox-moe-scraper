package catbox

import (
	"math/rand"
	"os"
	"strings"
)

func GenerateID() string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	const length = 6
	var id strings.Builder
	for i := 0; i < length; i++ {
		id.WriteByte(charset[rand.Intn(len(charset))])
	}
	return id.String()
}

func EnsureDownloadPathExists(downloadPath string) {
	if _, err := os.Stat(downloadPath); os.IsNotExist(err) {
		os.MkdirAll(downloadPath, os.ModePerm)
	}
}
