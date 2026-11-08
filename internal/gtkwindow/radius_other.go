//go:build !linux || (!amd64 && !arm64)

package gtkwindow

// Radius reports 0 where the window's shape is not read: macOS and Windows
// round the window and draw its decorations themselves, so the page has no
// shape of its own to follow.
func Radius() float32 { return 0 }
