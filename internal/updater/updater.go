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
		Name string `json:"name"`
		URL  string `json:"browser_download_url"`
	} `json:"assets"`
}

func CheckLatest(current string) (string, string, string, error) {
	resp, err := http.Get("https://api.github.com/repos/TenubahDEV/tenubah-agent/releases/latest")
	if err != nil {
		return "", "", "", err
	}
	defer resp.Body.Close()

	var r Release
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return "", "", "", err
	}

	if r.TagName == current {
		return "", "", "", nil
	}

	expected := binaryName()
	expectedSig := expected + ".sig"
	var binURL, sigURL string

	for _, a := range r.Assets {
		if a.Name == expected {
			binURL = a.URL
		}
		if a.Name == expectedSig {
			sigURL = a.URL
		}
	}

	if binURL == "" {
		return "", "", "", fmt.Errorf("binary %s not found", expected)
	}
	if sigURL == "" {
		return "", "", "", fmt.Errorf("signature %s not found", expectedSig)
	}

	return r.TagName, binURL, sigURL, nil
}

func binaryName() string {
	if runtime.GOOS == "windows" {
		return "tenubah-agent-windows-amd64.exe"
	}
	return "tenubah-agent-linux-amd64"
}
