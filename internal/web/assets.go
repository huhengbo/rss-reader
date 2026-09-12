package web

import "embed"

// Static contains the embedded frontend assets.
//
//go:embed static
var Static embed.FS
