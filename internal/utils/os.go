package utils

import (
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/eiannone/keyboard"
)

func PauseExit() {
	log.Println("Press any to exit")
	_, _, _ = keyboard.GetSingleKey()
	os.Exit(0)
}

func EXEName() string {
	path, err := Executable()
	if err != nil {
		return ""
	}
	return strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
}

func IsPortInUse(port int) bool {
	if conn, err := net.DialTimeout("tcp", net.JoinHostPort("", strconv.Itoa(port)), 3*time.Second); err == nil {
		conn.Close()
		return true
	}
	return false
}
