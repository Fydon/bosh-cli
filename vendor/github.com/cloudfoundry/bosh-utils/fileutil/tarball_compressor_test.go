package fileutil_test

import (
	"fmt"
	"os"
	"path/filepath"

	. "github.com/onsi/gomega"
)

func fixtureSrcDir() string {
	pwd, err := os.Getwd()
	Expect(err).NotTo(HaveOccurred())
	return filepath.Join(pwd, "test_assets", "test_filtered_copy_to_temp")
}

func fixtureSrcTgz() string {
	pwd, err := os.Getwd()
	Expect(err).NotTo(HaveOccurred())
	return filepath.Join(pwd, "test_assets", "compressor-decompress-file-to-dir.tgz")
}

func createTestSymlink() (string, error) {
	srcDir := fixtureSrcDir()
	symlinkPath := filepath.Join(srcDir, "symlink_dir")
	symlinkTarget := filepath.Join(srcDir, "../symlink_target")
	os.Remove(symlinkPath)
	return symlinkPath, os.Symlink(symlinkTarget, symlinkPath)
}

func beDir() beDirMatcher {
	return beDirMatcher{}
}

type beDirMatcher struct {
}

//FailureMessage(actual interface{}) (message string)
//NegatedFailureMessage(actual interface{}) (message string)
func (m beDirMatcher) Match(actual interface{}) (bool, error) {
	path, ok := actual.(string)
	if !ok {
		return false, fmt.Errorf("`%s' is not a valid path", actual)
	}

	dir, err := os.Open(path)
	if err != nil {
		return false, fmt.Errorf("Could not open `%s'", actual)
	}
	defer dir.Close()

	dirInfo, err := dir.Stat()
	if err != nil {
		return false, fmt.Errorf("Could not stat `%s'", actual)
	}

	return dirInfo.IsDir(), nil
}

func (m beDirMatcher) FailureMessage(actual interface{}) string {
	return fmt.Sprintf("Expected `%s' to be a directory", actual)
}

func (m beDirMatcher) NegatedFailureMessage(actual interface{}) string {
	return fmt.Sprintf("Expected `%s' to not be a directory", actual)
}
