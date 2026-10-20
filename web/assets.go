package web

import "embed"

// Files contains the production frontend built before compiling the executable.
//
//go:embed all:dist
var Files embed.FS
