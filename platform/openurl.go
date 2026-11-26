package platform

import (
	"errors"
	"os/exec"
	"runtime"
)

// openURLFallback shells out to the OS's URL handler. It is the default until a
// backend installs a better one: the SDL3 adapter replaces it with SDL_OpenURL,
// which also works where there is no xdg-open (Android and iOS).
func openURLFallback(url string) error {
	if url == "" {
		return errors.New("platform.OpenURL: empty url")
	}
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default: // linux and other unix-likes
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}
