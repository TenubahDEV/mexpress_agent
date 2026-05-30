package updater

import (
	"fmt"
	"io"
	"net/http"
	"os"
)

func Apply(url, sigURL string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	tmp := exe + ".new"
	sig := tmp + ".sig"

	// Ensure cleanup of previous temp files
	os.Remove(tmp)
	os.Remove(sig)

	if err := download(url, tmp); err != nil {
		return fmt.Errorf("failed to download binary: %w", err)
	}
	if err := download(sigURL, sig); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("failed to download signature: %w", err)
	}

	defer func() {
		os.Remove(tmp)
		os.Remove(sig)
	}()

	if err := VerifySignature(tmp, sig); err != nil {
		return fmt.Errorf("signature verification failed: %w", err)
	}

	// On Windows, a running executable file is locked and cannot be directly overwritten.
	// The standard way is to rename the currently running file to a different name (e.g. .old),
	// which Windows allows, and then move the new file to the original path.
	oldExe := exe + ".old"
	os.Remove(oldExe) // Remove old backup if it exists from a previous update

	if err := os.Rename(exe, oldExe); err != nil {
		// If renaming the running executable fails (e.g. on Linux where overwrite is permitted),
		// we fallback to direct overwrite.
		if err := os.Rename(tmp, exe); err != nil {
			return fmt.Errorf("failed to overwrite binary: %w", err)
		}
		return nil
	}

	// Place the new binary in the original executable's location
	if err := os.Rename(tmp, exe); err != nil {
		// If placing the new binary fails, try to restore the original binary
		os.Rename(oldExe, exe)
		return fmt.Errorf("failed to place new binary: %w", err)
	}

	// Try to remove the old binary. On Windows this might fail if the file is still held in memory,
	// which is fine and will be cleaned up on the next update attempt or OS reboot.
	os.Remove(oldExe)

	return nil
}

func download(url, path string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("received bad HTTP status code: %d", resp.StatusCode)
	}

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
