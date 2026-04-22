package helper

import (
	"path/filepath"
	"runtime"
)

func GetRootPath() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "..", "..")
}
