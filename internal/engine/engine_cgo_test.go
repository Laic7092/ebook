//go:build cgo

package engine

import (
	"strings"
	"testing"
)

const sample = `书名：测试书
作者：某人

第一章 开始

这是第一段。阅读器的排版必须能处理中文标点、段落缩进和换行。
这是第二段，用来把一页填满，好检查分页与翻页的边界。

第二章 继续

第二章的正文。这里的行数足够多，可以确认翻页不会越界，
也可以确认章节切换之后进度会重新从零开始。

第三章 结束

最后一段。
`

func openTest(t *testing.T) Engine {
	t.Helper()
	eng := New()
	if _, err := eng.Open("test.txt", []byte(sample)); err != nil {
		t.Fatalf("打开失败: %v", err)
	}
	t.Cleanup(func() { _ = eng.Close() })
	return eng
}

func TestOpenReportsBookAndTOC(t *testing.T) {
	eng := openTest(t)
	book, err := eng.Open("test.txt", []byte(sample))
	if err != nil {
		t.Fatalf("再次打开失败: %v", err)
	}
	if book.Title == "" {
		t.Fatalf("书名不该为空: %+v", book)
	}
	if len(book.Chapters) < 3 {
		t.Fatalf("章节数不对: %+v", book.Chapters)
	}
	if len(book.TOC) == 0 {
		t.Fatalf("目录不该为空")
	}
	for _, entry := range book.TOC {
		if entry.Spine < -1 {
			t.Fatalf("目录 spine 非法: %+v", entry)
		}
		if strings.TrimSpace(entry.Label) == "" {
			t.Fatalf("目录标题为空: %+v", entry)
		}
	}
}

func TestLayoutPagesAndBitmap(t *testing.T) {
	eng := openTest(t)
	viewport := Viewport{Width: 390, Height: 700, Top: 60, Right: 24, Bottom: 60, Left: 24}
	pages, err := eng.Layout(0, viewport, Style{FontScale: 1, LineScale: 1})
	if err != nil {
		t.Fatalf("排版失败: %v", err)
	}
	if len(pages) == 0 {
		t.Fatalf("没有页")
	}
	first := pages[0]
	if first.LineLen <= 0 {
		t.Fatalf("第一页没有行: %+v", first)
	}
	// 内容盒 = 视口 - 内边距。
	if want := float32(390 - 48); first.Width != want {
		t.Fatalf("内容盒宽度 = %v, 期望 %v", first.Width, want)
	}
	// 页号顺序即下标，进度单调不减。
	for i := 1; i < len(pages); i++ {
		if pages[i].Progress < pages[i-1].Progress {
			t.Fatalf("进度不单调: %v -> %v", pages[i-1].Progress, pages[i].Progress)
		}
	}

	bitmap, err := eng.Bitmap(0, 2, Style{})
	if err != nil {
		t.Fatalf("光栅化失败: %v", err)
	}
	if bitmap.W != 780 || bitmap.H != 1400 {
		t.Fatalf("位图尺寸 = %dx%d, 期望 780x1400", bitmap.W, bitmap.H)
	}
	if len(bitmap.RGBA) != bitmap.W*bitmap.H*4 {
		t.Fatalf("像素长度不对: %d", len(bitmap.RGBA))
	}
	// 白纸：没有字体时也应该是纯白，而不是未初始化的内存。
	if bitmap.RGBA[0] != 0xff || bitmap.RGBA[1] != 0xff {
		t.Fatalf("纸色不对: %v", bitmap.RGBA[:4])
	}
	if _, err := eng.Bitmap(9999, 1, Style{}); err == nil {
		t.Fatalf("越界页号应该报错")
	}
}

func TestHitProgressSelection(t *testing.T) {
	eng := openTest(t)
	viewport := Viewport{Width: 390, Height: 700, Top: 60, Right: 24, Bottom: 60, Left: 24}
	if _, err := eng.Layout(0, viewport, Style{}); err != nil {
		t.Fatalf("排版失败: %v", err)
	}
	hit, ok := eng.Hit(0, 100, 120)
	if !ok {
		t.Fatalf("正文区域应该能命中")
	}
	if hit.Page != 0 {
		t.Fatalf("命中页号 = %d", hit.Page)
	}
	progress, ok := eng.Progress(hit)
	if !ok || progress < 0 || progress > 1 {
		t.Fatalf("进度非法: %v %v", progress, ok)
	}
	sel, ok := eng.Selection(hit, Location{Page: hit.Page, Line: hit.Line, Byte: hit.Byte + 4})
	if !ok {
		t.Fatalf("选区应该成功")
	}
	if sel.Text == "" {
		t.Fatalf("选区没有原文")
	}
	if len(sel.Rects) == 0 {
		t.Fatalf("选区没有矩形")
	}
}

func TestStyleRepaginates(t *testing.T) {
	eng := openTest(t)
	viewport := Viewport{Width: 390, Height: 700, Top: 60, Right: 24, Bottom: 60, Left: 24}
	small, err := eng.Layout(0, viewport, Style{FontScale: 1, LineScale: 1})
	if err != nil {
		t.Fatalf("排版失败: %v", err)
	}
	big, err := eng.Layout(0, viewport, Style{FontScale: 1.6, LineScale: 1.5})
	if err != nil {
		t.Fatalf("排版失败: %v", err)
	}
	if len(big) < len(small) {
		t.Fatalf("放大字号应该变多页: %d -> %d", len(small), len(big))
	}
	pages := eng.Pages()
	if len(pages) != len(big) {
		t.Fatalf("Pages() 与 Layout() 不一致: %d vs %d", len(pages), len(big))
	}
}

func TestBaseFontSizeAndTOC(t *testing.T) {
	eng := openTest(t)
	if size := eng.BaseFontSize(); size <= 0 {
		t.Fatalf("基准字号非法: %v", size)
	}
	if len(eng.TOC()) == 0 {
		t.Fatalf("TOC() 为空")
	}
	if eng.FontFaces() != 0 {
		t.Logf("文本格式不该有内嵌字体，但返回了 %d", eng.FontFaces())
	}
}
