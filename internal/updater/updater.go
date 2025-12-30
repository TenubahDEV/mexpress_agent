package updater

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
)

type Release struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

func CheckLatest(current string) (string, string, error) {
	resp, err := http.Get("https://api.github.com/repos/TenubahDEV/tenubah-agent/releases/latest")
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	var r Release
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return "", "", err
	}

	if r.TagName == current {
		return "", "", nil
	}

	binName := binaryName()
	for _, a := range r.Assets {
		if a.Name == binName {
			return r.TagName, a.BrowserDownloadURL, nil
		}
	}

	return "", "", fmt.Errorf("binary not found for %s", binName)
}

func binaryName() string {
	if runtime.GOOS == "windows" {
		return "tenubah-agent-windows-amd64.exe"
	}
	return "tenubah-agent-linux-amd64"
}
