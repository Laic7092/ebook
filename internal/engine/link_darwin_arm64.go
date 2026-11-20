//go:build cgo && darwin && arm64

package engine

// #cgo LDFLAGS: ${SRCDIR}/native/darwin_arm64/libebook_abi.a -framework Security
import "C"
