package updater

import (
	"io"
	"net/http"
	"os"
)

func ApplyUpdate(url string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}

	tmp := exe + ".new"

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(tmp)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		return err
	}

	return os.Rename(tmp, exe)
}
