package models

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func autologin() {
	initChrome()
	initWebDisplay()

}

func initChrome() {
	var chromePath string
	if runtime.GOOS == "windows" {
		chromePath = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Local", "pyppeteer", "pyppeteer", "local-chromium", "588429", "chrome-win32", "chrome.exe")
	} else if runtime.GOOS == "linux" {
		chromePath = filepath.Join(os.Getenv("HOME"), ".local", "share", "pyppeteer", "local-chromium", "1181205", "chrome-linux", "chrome")
	} else if runtime.GOOS == "darwin" {
		fmt.Println("mac")
		return
	} else {
		fmt.Println("unknown")
		return
	}

	if _, err := os.Stat(chromePath); os.IsNotExist(err) {
		fmt.Println("Chrome is not installed")

		//todo Add your code to download and install Chrome
	} else {
		fmt.Println("Chrome is installed")
	}
}

var WebDisplay = true
var configfile = "config.ini"

func initWebDisplay() {
	file, err := os.Open(configfile)
	if err != nil {
		fmt.Println("读取配置文件时出错")
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "Displaylogin=1") {
			WebDisplay = false
			fmt.Println("当前模式：显示web登录图形化界面")
			break
		}
	}

	if WebDisplay {
		fmt.Println("当前配置不显示web登录图形化界面，若要取消静默登陆，在配置文件中设置参数Displaylogin=1")
	}
}
