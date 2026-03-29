package server

import (
	"embed"
	"fmt"
)

//go:embed resources/themes/*.css
var themeFS embed.FS

// LoadTheme returns the combined base + theme CSS for the given theme name.
// Valid themes: classic, terminal, modern, daily, raw, win.
// An empty name defaults to "classic".
func LoadTheme(name string) (string, error) {
	if name == "" {
		name = "terminal"
	}

	base, err := themeFS.ReadFile("resources/themes/base.css")
	if err != nil {
		return "", fmt.Errorf("reading base.css: %w", err)
	}

	theme, err := themeFS.ReadFile("resources/themes/" + name + ".css")
	if err != nil {
		return "", fmt.Errorf("unknown theme %q: %w", name, err)
	}

	return string(theme) + "\n" + string(base), nil
}
