package ui

// fakeEngine 是测试用的引擎：不解析文件，只按 viewport 生成页数，
// 位图是纸色底加几条灰线和页码块。它记录调用次数，让测试能断言
// “字号变了会重新排版”。
import (
	"fmt"
	"sync"

	. "github.com/egoist/mygo/ui"
)

type fakeEngine struct {
	mu sync.Mutex

	book  Book
	base  float32
	spine int

	layout     Layout
	viewport   Rect
	layoutOK   bool
	callLayout int
	callBitmap int
}

func newFakeEngine() *fakeEngine {
	book := Book{
		Title:  "测试之书",
		Author: "某人",
		Chapters: []Chapter{
			{Spine: 0, Title: "第一章 起"},
			{Spine: 1, Title: "第二章 承"},
			{Spine: 2, Title: "第三章 合"},
		},
		TOC: []TOCEntry{
			{Label: "第一章 起", Spine: 0},
			{Label: "第一节 深处", Spine: 0, Fragment: "deep", Depth: 1},
			{Label: "第二章 承", Spine: 1, Fragment: "c2"},
			{Label: "第三章 合", Spine: 2, Fragment: "c3", Depth: 1},
		},
	}
	return &fakeEngine{book: book, base: 16}
}

func (f *fakeEngine) Open(name string, data []byte) (Book, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.book, nil
}

func (f *fakeEngine) Layout(spine int, viewport Rect, style Style) (Layout, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.callLayout++
	f.spine = spine
	f.viewport = viewport
	f.layoutOK = true

	scale := style.FontScale
	if scale <= 0 {
		scale = 1
	}
	lineScale := style.LineScale
	if lineScale <= 0 {
		lineScale = 1
	}
	lineH := 20 * scale * lineScale
	if style.Bold {
		lineH++
	}
	per := int(viewport.H / lineH)
	if per < 1 {
		per = 1
	}
	total := 70 + spine*20 // 每章行数不同，页数也不同
	pages := (total + per - 1) / per
	if pages < 1 {
		pages = 1
	}
	out := Layout{Spine: spine, Pages: make([]PageMeta, pages)}
	for i := range out.Pages {
		from := i * per
		n := min(per, total-from)
		out.Pages[i] = PageMeta{
			Box:      Rect{W: viewport.W, H: viewport.H},
			LineFrom: from,
			LineLen:  n,
			Progress: float32(from+n) / float32(total),
		}
	}
	f.layout = out
	return out, nil
}

func (f *fakeEngine) Bitmap(page int, scale float32, style Style) (Raster, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.callBitmap++
	if !f.layoutOK {
		return Raster{}, fmt.Errorf("fake: bitmap before layout")
	}
	if page < 0 || page >= len(f.layout.Pages) {
		return Raster{}, fmt.Errorf("fake: page %d out of range", page)
	}
	if scale <= 0 {
		scale = 1
	}
	w := int(f.viewport.W * scale)
	h := int(f.viewport.H * scale)
	if w < 1 || h < 1 {
		return Raster{}, fmt.Errorf("fake: empty page")
	}
	bg := argb(style.Paper, 0xfffbfaf7)
	ink := argb(style.Ink, 0xff20201e)
	pix := make([]byte, w*h*4)
	for i := 0; i < len(pix); i += 4 {
		pix[i], pix[i+1], pix[i+2], pix[i+3] = bg[0], bg[1], bg[2], 255
	}
	// 几行“正文”，让页面看起来不是纯色。
	lineY := h / 5
	for n := 0; n < 10 && lineY < h*4/5; n++ {
		x0, x1 := w/8, w*7/8
		if n == 9 {
			x1 = w / 2
		}
		fillRect(pix, w, x0, lineY, x1, lineY+max(1, h/120), grey(ink, 0.22))
		lineY += max(2, h/28)
	}
	drawPageNumber(pix, w, h, page+1, ink)
	return Raster{W: w, H: h, RGBA: pix}, nil
}

func (f *fakeEngine) Hit(page int, x, y float32) (Location, bool) {
	return Location{Page: uint32(page), Line: uint32(y / 20)}, true
}

func (f *fakeEngine) LinkAt(page int, x, y float32) (string, bool) { return "", false }

func (f *fakeEngine) Selection(a, b Location) (TextSelection, bool) { return TextSelection{}, false }

func (f *fakeEngine) Progress(loc Location) (float32, bool) { return 0, true }

func (f *fakeEngine) FragmentLine(fragment string) (int, bool) {
	switch fragment {
	case "c2":
		return 8, true
	case "c3":
		return 12, true
	case "deep":
		return 90, true
	}
	return 0, false
}

func (f *fakeEngine) ResolveLink(href string) (EngineLink, bool) { return EngineLink{}, false }

func (f *fakeEngine) RegisterFont(data []byte) (int, error) { return 0, nil }

func (f *fakeEngine) BaseFontSize() float32 {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.base
}

func (f *fakeEngine) TOC() []TOCEntry {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]TOCEntry(nil), f.book.TOC...)
}

func (f *fakeEngine) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	return nil
}

func (f *fakeEngine) layoutCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.callLayout
}

// argb 把 0xRRGGBBAA 拆成 RGB，0 时用 fallback 的 RGB。
func argb(v uint32, fallback uint32) [3]uint8 {
	if v == 0 {
		v = fallback
	}
	return [3]uint8{uint8(v >> 24), uint8(v >> 16), uint8(v >> 8)}
}

func grey(c [3]uint8, k float32) [3]uint8 {
	mix := func(v uint8) uint8 { return uint8(float32(v) * k) }
	return [3]uint8{mix(c[0]), mix(c[1]), mix(c[2])}
}

func fillRect(pix []byte, w, x0, y0, x1, y1 int, c [3]uint8) {
	if x0 < 0 {
		x0 = 0
	}
	if y0 < 0 {
		y0 = 0
	}
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			i := (y*w + x) * 4
			if i < 0 || i+3 >= len(pix) {
				continue
			}
			pix[i], pix[i+1], pix[i+2], pix[i+3] = c[0], c[1], c[2], 255
		}
	}
}

// 3x5 点阵数字，够画页码了。
var digitGlyphs = [10][5]uint8{
	{0b111, 0b101, 0b101, 0b101, 0b111},
	{0b010, 0b110, 0b010, 0b010, 0b111},
	{0b111, 0b001, 0b111, 0b100, 0b111},
	{0b111, 0b001, 0b111, 0b001, 0b111},
	{0b101, 0b101, 0b111, 0b001, 0b001},
	{0b111, 0b100, 0b111, 0b001, 0b111},
	{0b111, 0b100, 0b111, 0b101, 0b111},
	{0b111, 0b001, 0b010, 0b010, 0b010},
	{0b111, 0b101, 0b111, 0b101, 0b111},
	{0b111, 0b101, 0b111, 0b001, 0b111},
}

func drawPageNumber(pix []byte, w, h, n int, ink [3]uint8) {
	digits := []int{}
	for n > 0 {
		digits = append([]int{n % 10}, digits...)
		n /= 10
	}
	if len(digits) == 0 {
		digits = []int{0}
	}
	scale := max(2, w/220)
	dotW, dotH, gap := 3*scale, 5*scale, scale
	totalW := len(digits)*dotW + (len(digits)-1)*gap
	x0 := (w - totalW) / 2
	y0 := h / 12
	for _, d := range digits {
		g := digitGlyphs[d]
		for row := 0; row < 5; row++ {
			for col := 0; col < 3; col++ {
				if g[row]&(1<<(2-col)) == 0 {
					continue
				}
				fillRect(pix, w, x0+col*scale, y0+row*scale,
					x0+(col+1)*scale, y0+(row+1)*scale, ink)
			}
		}
		x0 += dotW + gap
	}
	_ = dotH
}
