package cmdutil

import (
	"os/exec"
	"runtime"
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
