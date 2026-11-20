//go:build cgo

package engine

/*
#include <stdint.h>
#include <stddef.h>
#include <stdlib.h>

typedef struct EbEngine EbEngine;

extern void *eb_alloc(size_t len);
extern void eb_free(void *ptr, size_t len);
extern EbEngine *eb_open(const uint8_t *bytes, size_t len, const char *name);
extern void eb_close(EbEngine *engine);
extern uint8_t *eb_request(EbEngine *engine, const uint8_t *req, size_t len, size_t *out_len);
extern const uint8_t *eb_last_error(size_t *len);
*/
import "C"

import (
	"fmt"
	"sync"
	"unsafe"
)

type cgo struct {
	mu  sync.Mutex
	ptr *C.EbEngine
}

// New 返回一个空的会话；`Open` 之后才能排版。
func New() Engine { return &cgo{} }

func (e *cgo) call(request []byte) (response, error) {
	if e.ptr == nil {
		return response{}, fmt.Errorf("engine: 还没有打开书")
	}
	if len(request) == 0 {
		return response{}, fmt.Errorf("engine: 空请求")
	}
	var outLen C.size_t
	ptr := C.eb_request(
		e.ptr,
		(*C.uint8_t)(unsafe.Pointer(&request[0])),
		C.size_t(len(request)),
		&outLen,
	)
	if ptr == nil {
		return response{}, fmt.Errorf("engine: 调用失败")
	}
	defer C.eb_free(unsafe.Pointer(ptr), outLen)
	data := C.GoBytes(unsafe.Pointer(ptr), C.int(outLen))
	got, err := decodeResponse(data)
	if err != nil {
		return response{}, err
	}
	if got.kind == respError {
		return response{}, fmt.Errorf("engine: %s", got.message)
	}
	return got, nil
}

func (e *cgo) Open(name string, data []byte) (Book, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.ptr != nil {
		C.eb_close(e.ptr)
		e.ptr = nil
	}
	if len(data) == 0 {
		return Book{}, fmt.Errorf("engine: 空的书籍数据")
	}
	cname := C.CString(name)
	defer C.free(unsafe.Pointer(cname))
	ptr := C.eb_open(
		(*C.uint8_t)(unsafe.Pointer(&data[0])),
		C.size_t(len(data)),
		cname,
	)
	if ptr == nil {
		var errLen C.size_t
		message := C.eb_last_error(&errLen)
		return Book{}, fmt.Errorf("engine: 打开失败: %s", C.GoStringN((*C.char)(unsafe.Pointer(message)), C.int(errLen)))
	}
	e.ptr = ptr

	got, err := e.call(encodeSimple(reqBook))
	if err != nil {
		return Book{}, err
	}
	if got.book == nil {
		return Book{}, errNoValue
	}
	toc, err := e.call(encodeSimple(reqTOC))
	if err != nil {
		return Book{}, err
	}
	book := *got.book
	book.TOC = toc.toc
	return book, nil
}

func (e *cgo) Layout(spine int, viewport Viewport, style Style) ([]PageMeta, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	got, err := e.call(encodeLayout(spine, viewport, style))
	if err != nil {
		return nil, err
	}
	return got.pages, nil
}

func (e *cgo) Bitmap(page int, scale float32, style Style) (Bitmap, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	got, err := e.call(encodeBitmap(page, scale, style))
	if err != nil {
		return Bitmap{}, err
	}
	if got.bitmap == nil {
		return Bitmap{}, errNoValue
	}
	return *got.bitmap, nil
}

func (e *cgo) Hit(page int, x, y float32) (Location, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	got, err := e.call(encodePoint(reqHit, page, x, y))
	if err != nil || got.hit == nil {
		return Location{}, false
	}
	return *got.hit, true
}

func (e *cgo) LinkAt(page int, x, y float32) (string, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	got, err := e.call(encodePoint(reqLinkAt, page, x, y))
	if err != nil || got.link == nil {
		return "", false
	}
	return *got.link, true
}

func (e *cgo) Selection(a, b Location) (Selection, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	got, err := e.call(encodeSelection(a, b))
	if err != nil || got.selection == nil {
		return Selection{}, false
	}
	return *got.selection, true
}

func (e *cgo) Progress(loc Location) (float32, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	got, err := e.call(encodeLocRequest(reqProgress, loc))
	if err != nil || got.progress == nil {
		return 0, false
	}
	return *got.progress, true
}

func (e *cgo) FragmentLine(fragment string) (int, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	got, err := e.call(encodeStringRequest(reqFragmentLine, fragment))
	if err != nil || got.line == nil {
		return 0, false
	}
	return *got.line, true
}

func (e *cgo) ResolveLink(href string) (Link, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	got, err := e.call(encodeStringRequest(reqResolveLink, href))
	if err != nil || got.target == nil {
		return Link{}, false
	}
	return *got.target, true
}

func (e *cgo) RegisterFont(data []byte) (int, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	got, err := e.call(encodeFont(data))
	if err != nil {
		return 0, err
	}
	return int(got.font), nil
}

func (e *cgo) RegisterFontFace(index int) (int, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	got, err := e.call(encodeFontFace(index))
	if err != nil {
		return 0, err
	}
	return int(got.font), nil
}

func (e *cgo) FontFaces() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	got, err := e.call(encodeSimple(reqBook))
	if err != nil || got.book == nil {
		return 0
	}
	return len(got.book.Fonts)
}

func (e *cgo) BaseFontSize() float32 {
	e.mu.Lock()
	defer e.mu.Unlock()
	got, err := e.call(encodeSimple(reqBaseFontSize))
	if err != nil {
		return 16
	}
	return got.number
}

func (e *cgo) TOC() []TOCEntry {
	e.mu.Lock()
	defer e.mu.Unlock()
	got, err := e.call(encodeSimple(reqTOC))
	if err != nil {
		return nil
	}
	return got.toc
}

func (e *cgo) Pages() []PageMeta {
	e.mu.Lock()
	defer e.mu.Unlock()
	got, err := e.call(encodeSimple(reqPages))
	if err != nil {
		return nil
	}
	return got.pages
}

func (e *cgo) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.ptr != nil {
		C.eb_close(e.ptr)
		e.ptr = nil
	}
	return nil
}
