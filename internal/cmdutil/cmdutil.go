package cmdutil

import (
	"os"
	"os/exec"
	"runtime"
	"strings"

	"golang.org/x/crypto/ssh/terminal"
)

type URLOpener interface {
	Open(url string) error
}

type Browser struct{}

func (b *Browser) Open(url string) error {
	browser := searchBrowserLauncher(runtime.GOOS)
	c := exec.Command(browser, url)
	if err := c.Run(); err != nil {
		return err
	}
	return nil
}

func searchBrowserLauncher(goos string) (browser string) {
	switch goos {
	case "darwin":
		browser = "open"
	case "windows":
		browser = "cmd /c start"
	default:
		candidates := []string{
			"xdg-open",
			"cygstart",
			"x-www-browser",
			"firefox",
			"opera",
			"mozilla",
			"netscape",
		}
		for _, b := range candidates {
			path, err := exec.LookPath(b)
			if err == nil {
				browser = path
				break
			}
		}
	}
	return browser
}

func IsOverScreeenRow(contents string) bool {
	row := strings.Count(contents, "\n")
	_, height, _ := terminal.GetSize(0)
	if row > height {
		return true
	}
	return false
}

func ShowPager(contents string) error {
	cmd := exec.Command("less", "-R")
	cmd.Stdin = strings.NewReader(contents)
	cmd.Stdout = os.Stdout
	err := cmd.Run()
	if err != nil {
		return err
	}
	return nil
}
