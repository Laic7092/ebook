//go:build cgo && windows && amd64

package engine

// #cgo LDFLAGS: ${SRCDIR}/native/windows_amd64/libebook_abi.a -lws2_32 -luserenv -lbcrypt -lntdll
import "C"
