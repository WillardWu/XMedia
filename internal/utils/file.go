package utils

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
)

func FileTotalPath(name string) string {
	return filepath.Join(CWD(), name)
}

func CWD() string {
	path, err := Executable()
	if err != nil {
		return ""
	}
	return filepath.Dir(path)
}

func Executable() (string, error) {
	path, err := os.Executable()
	if err != nil {
		file, err := exec.LookPath(os.Args[0])
		if err != nil {
			return "", err
		}
		path, err = filepath.Abs(file)
		if err != nil {
			return "", err
		}
	}
	return path, nil
}

func Exist(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}
	return true
}

func EnsureDir(dir string) (err error) {
	if _, err = os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0755)
		if err != nil {
			return
		}
	}
	return
}

func DataDir() string {
	dir := CWD()
	dir = ExpandHomeDir(dir)
	if err := EnsureDir(dir); err != nil {
		log.Printf("ensure data dir %s error, %v", dir, err)
		return ""
	}
	return dir
}

func ExpandHomeDir(path string) string {
	if len(path) == 0 {
		return path
	}
	if path[0] != '~' {
		return path
	}
	if len(path) > 1 && path[1] != '/' && path[1] != '\\' {
		return path
	}
	return filepath.Join(HomeDir(), path[1:])
}

func HomeDir() string {
	u, err := user.Current()
	if err != nil {
		return ""
	}
	return u.HomeDir
}

func GenerateUniqueString16() (string, error) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
