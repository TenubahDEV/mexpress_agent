package updater

import (
	"io"
	"net/http"
	"os"
)

func Apply(url, sigURL string) error {
	exe, _ := os.Executable()
	tmp := exe + ".new"
	sig := tmp + ".sig"

	download(url, tmp)
	download(sigURL, sig)

	if err := VerifySignature(tmp, sig); err != nil {
		return err
	}

	return os.Rename(tmp, exe)
}

func download(url, path string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		return err
	}

	return nil
}
