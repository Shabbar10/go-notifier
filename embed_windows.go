package notifier

import (
	"crypto/md5"
	_ "embed"
	"encoding/hex"
	"os"
	"path/filepath"
	"sync"
)

//go:embed bin/toaster.exe
var toasterExe []byte

var (
	toasterPath     string
	toasterExtract  sync.Once
	toasterErr      error
)

func getToasterPath() (string, error) {
	toasterExtract.Do(func() {
		hash := md5.Sum(toasterExe)
		hashStr := hex.EncodeToString(hash[:])
		toasterPath = filepath.Join(os.TempDir(), "synchronum_toaster_"+hashStr+".exe")

		if _, err := os.Stat(toasterPath); err == nil {
			return
		}

		toasterErr = os.WriteFile(toasterPath, toasterExe, 0755)
	})
	return toasterPath, toasterErr
}
