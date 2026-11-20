package ui

import (
	"fmt"
	"math"
)

// fakeEngine 是测试用的引擎：按 viewport 和字号算出页数，位图是纸色底 +
// 页码块 + 若干行文字块。它不碰内部 engine 包，只实现 ui.Engine。
type fakeEngine struct {
	book        Book
	layoutCalls int
	bitmapCalls int
	closed      bool

	lastSpine int
	lastVP    Viewport
	lastStyle Style
}

func newFakeEngine() *fakeEngine {
	return &fakeEngine{book: Book{
		Title:  "测试之书",
		Author: "测试作者",
		Chapters: []Chapter{
			{Spine: 0}, {Spine: 1}, {Spine: 2},
		},
		TOC: []TOCEntry{
			{Label: "第一章 起风", Spine: 0},
			{Label: "第二章 下雨", Spine: 1, Fragment: "ch2"},
			{Label: "第三章 放晴", Spine: 2, Depth: 1},
		},
	}}
}

func (f *fakeEngine) Open(name string, data []byte) (Book, error) {
	if len(data) == 0 && name == "" {
		return Book{}, fmt.Errorf("空的输入")
	}
	return f.book, nil
}

func (f *fakeEngine) Layout(spine int, viewport Viewport, style Style) ([]PageMeta, error) {
	f.layoutCalls++
	f.lastSpine, f.lastVP, f.lastStyle = spine, viewport, style
	fs := style.FontScale
	if fs <= 0 {
		fs = 1
	}
	ls := style.LineScale
	if ls <= 0 {
		ls = 1
	}
	w, h := maxf(1, viewport.Width), maxf(1, viewport.Height)
	// 面积越大页数越少；字号/行距越大页数越多。至少两页，方便测翻页。
	n := int(w*h/(19600*fs*ls)) / 4
	n = clampInt(n, 2, 40)
	pages := make([]PageMeta, n)
	for i := range pages {
		pages[i] = PageMeta{
			Width:    viewport.Width,
			Height:   viewport.Height,
			LineFrom: i * 20,
			LineLen:  20,
			Progress: float32(i+1) / float32(n),
		}
	}
	return pages, nil
}

func (f *fakeEngine) Bitmap(page int, scale float32, style Style) (Bitmap, error) {
	f.bitmapCalls++
	if scale <= 0 {
		scale = 1
	}
	w := int(math.Round(float64(maxf(1, f.lastVP.Width) * scale)))
	h := int(math.Round(float64(maxf(1, f.lastVP.Height) * scale)))
	if w <= 0 || h <= 0 {
		w, h = 100, 140
	}
	paper := uint32(0xffffffff)
	if style.Paper != 0 {
		paper = style.Paper
	}
	ink := uint32(0x222222ff)
	if style.Ink != 0 {
		ink = style.Ink
	}
	rgba := make([]byte, w*h*4)
	for i := 0; i < w*h; i++ {
		rgba[i*4+0] = uint8(paper >> 24)
		rgba[i*4+1] = uint8(paper >> 16)
		rgba[i*4+2] = uint8(paper >> 8)
		rgba[i*4+3] = 255
	}
	fill := func(x, y, ww, hh int, c uint32) {
		for yy := maxInt(0, y); yy < minInt(h, y+hh); yy++ {
			for xx := maxInt(0, x); xx < minInt(w, x+ww); xx++ {
				i := (yy*w + xx) * 4
				rgba[i+0] = uint8(c >> 24)
				rgba[i+1] = uint8(c >> 16)
				rgba[i+2] = uint8(c >> 8)
				rgba[i+3] = 255
			}
		}
	}
	// 页码块：第 n 页画 n 个小方块，测试不依赖字形。
	for i := 0; i <= page && i < 12; i++ {
		fill(w/2-(page+1)*8+i*16, h/2-16, 10, 32, ink)
	}
	// 几行“正文”块。
	lineH := maxInt(2, int(14*scale))
	for row := 0; row < 8; row++ {
		y := h/4 + row*lineH*2
		if y+lineH >= h {
			break
		}
		fill(w/8, y, w-w/4, lineH, ink)
	}
	return Bitmap{W: w, H: h, RGBA: rgba}, nil
}

func (f *fakeEngine) Hit(page int, x, y float32) (Location, bool) {
	return Location{Page: uint32(page)}, true
}
func (f *fakeEngine) LinkAt(page int, x, y float32) (string, bool) { return "", false }
func (f *fakeEngine) Selection(a, b Location) (Selection, bool)    { return Selection{}, false }
func (f *fakeEngine) Progress(loc Location) (float32, bool)        { return 0, false }
func (f *fakeEngine) FragmentLine(fragment string) (int, bool) {
	if fragment == "ch2" {
		return 25, true
	}
	return 0, false
}
func (f *fakeEngine) ResolveLink(href string) (Link, bool)  { return Link{}, false }
func (f *fakeEngine) RegisterFont(data []byte) (int, error) { return 1, nil }
func (f *fakeEngine) BaseFontSize() float32                 { return 16 }
func (f *fakeEngine) TOC() []TOCEntry                       { return f.book.TOC }
func (f *fakeEngine) RegisterFontFace(index int) (int, error) { return 1, nil }
func (f *fakeEngine) FontFaces() int                          { return 0 }
func (f *fakeEngine) Pages() []PageMeta                       { return nil }
func (f *fakeEngine) Close() error {
	f.closed = true
	return nil
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
