package auth

import (
	"os/exec"
	"runtime"
)

// OpenBrowserFunc can be overridden in tests to prevent opening a real browser.
var OpenBrowserFunc = openBrowserDefault

func openBrowser(url string) error {
	return OpenBrowserFunc(url)
}

func openBrowserDefault(url string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", url).Start()
	case "linux":
		return exec.Command("xdg-open", url).Start()
	default:
		return exec.Command("open", url).Start()
	}
}
