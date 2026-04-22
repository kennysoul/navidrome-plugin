//go:build !wasip1

package main

// Stub for non-WASM builds (e.g., local go build / IDE support).
// The actual plugin logic lives in main.go (wasip1 build tag).

func main() {}
