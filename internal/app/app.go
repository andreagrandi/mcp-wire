package app

const (
	Name    = "mcp-wire"
	Author  = "Andrea Grandi"
	License = "MIT"
)

var Version = "0.1.1"

type App struct {
	Name    string
	Version string
	Author  string
	License string
}

func New() *App {
	return &App{
		Name:    Name,
		Version: Version,
		Author:  Author,
		License: License,
	}
}

func (a *App) GetFullVersion() string {
	return a.Name + " version " + a.Version
}
