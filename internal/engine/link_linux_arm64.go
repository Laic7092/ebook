//go:build cgo && linux && arm64

package engine

// #cgo LDFLAGS: ${SRCDIR}/native/linux_arm64/libebook_abi.a -lm -lpthread -ldl
import "C"
