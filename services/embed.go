package services

import "embed"

// FS contains bundled service definitions shipped with the binary.
//
//go:embed *.yaml
var FS embed.FS
