//go:build cgo && ios && arm64

package engine

// #cgo LDFLAGS: ${SRCDIR}/native/ios_arm64/libebook_abi.a -framework Security
import "C"
