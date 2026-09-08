// Package hiragana5 embeds the reviewed v1 curriculum pack files.
package hiragana5

import "embed"

// V1 is the published content tree (manifest, characters, strokes, traces, review).
//
//go:embed all:v1
var V1 embed.FS
