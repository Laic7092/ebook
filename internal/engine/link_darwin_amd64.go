//go:build cgo && darwin && amd64

package engine

// #cgo LDFLAGS: ${SRCDIR}/native/darwin_amd64/libebook_abi.a -framework Security
import "C"
