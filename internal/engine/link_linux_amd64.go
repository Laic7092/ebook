//go:build cgo && linux && amd64

package engine

// #cgo LDFLAGS: ${SRCDIR}/native/linux_amd64/libebook_abi.a -lm -lpthread -ldl
import "C"
