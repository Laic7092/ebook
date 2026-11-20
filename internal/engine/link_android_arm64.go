//go:build cgo && android && arm64

package engine

// #cgo LDFLAGS: ${SRCDIR}/native/android_arm64/libebook_abi.a -llog -ldl -lm
import "C"
