package grafanasdkclistarter

import (
	"fmt"
	"os"
	"strings"
)

func GetFlagEnvByFlagName(flagName, appName string) string {
	return fmt.Sprintf("%s_%s", appName, strings.ToUpper(flagName))
}

// EnsureDir checks if given directory exist, creates if not
func EnsureDir(dir string) error {
	if !DirExist(dir) {
		err := os.Mkdir(dir, 0755)
		if err != nil {
			return err
		}
	}
	return nil
}

// DirExist checks if directory exist
func DirExist(dir string) bool {
	_, err := os.Stat(dir)
	if err == nil {
		return true
	}
	return !os.IsNotExist(err)
}
