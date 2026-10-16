// Package runtimeassets embeds locally exported, code-free game resources.
package runtimeassets

import "embed"

// The placeholder keeps a clean source checkout buildable before extraction.
//
//go:embed *
var Files embed.FS
