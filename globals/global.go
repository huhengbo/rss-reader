package globals

import "embed"

// DirStatic contains the embedded frontend assets.
// Runtime configuration and mutable application state live in explicit components.
//
//go:embed static
var DirStatic embed.FS
