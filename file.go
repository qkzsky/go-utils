package utils

import (
	"os"
	"strings"
)

func FileExists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

func ParentDirectory(directory string) string {
	return Substr(directory, 0, strings.LastIndex(directory, string(os.PathSeparator)))
}
