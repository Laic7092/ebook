// Package reader 把 internal/engine 的引擎实现接到 ui 包依赖的 ui.Engine 上。
//
// ui 包只认 docs/engine-api.md 描述的那份接口，不 import internal/engine，
// 这样视图和测试都不需要 cgo；真正的引擎在这里转接一次，两边的差异只有：
//
//   - ui.Layout 是「一章 + 页列表」，引擎给的是 []PageMeta，这里补回 spine；
//   - ui.Layout 的 viewport 是页面矩形（点），引擎要的是带内边距的 Viewport；
//   - ui.Raster / ui.TextSelection / ui.EngineLink 是同结构改名；
//   - ui.Chapter 有 Title，引擎的 Chapter 没有，标题从 TOC 里按 spine 补。
package reader

import (
	"github.com/Laic7092/ebook/internal/engine"
	"github.com/Laic7092/ebook/ui"
	m "github.com/egoist/mygo/ui"
)

// Bridge 是 ui.Engine 在真引擎上的实现。它不是并发安全的，调用方（ui.App
// 的那把锁）负责串行化。
type Bridge struct {
	e engine.Engine
}

// New 用引擎实现造一个给 ui 用的适配器。
func New(e engine.Engine) ui.Engine { return &Bridge{e: e} }

// insetRatio 把页面矩形按比例切成内边距：左右 6%，上下 5%。
// 和页面几何一起缩放，所以手机上不会出现「正文贴边」或「留白过多」。
const (
	insetX = 0.06
	insetY = 0.05
)

func (b *Bridge) Open(name string, data []byte) (ui.Book, error) {
	book, err := b.e.Open(name, data)
	if err != nil {
		return ui.Book{}, err
	}

	// TOC 的一份副本，接口返回给调用方的切片不和引擎共享。
	toc := make([]ui.TOCEntry, len(book.TOC))
	titles := make(map[int]string, len(book.TOC))
	for i, e := range book.TOC {
		toc[i] = ui.TOCEntry{Label: e.Label, Spine: e.Spine, Fragment: e.Fragment, Depth: e.Depth}
		if e.Spine >= 0 && e.Label != "" {
			if _, ok := titles[e.Spine]; !ok { // 取最深层次里的第一条
				titles[e.Spine] = e.Label
			}
		}
	}

	chapters := make([]ui.Chapter, len(book.Chapters))
	for i, c := range book.Chapters {
		chapters[i] = ui.Chapter{Spine: c.Spine, Title: titles[c.Spine]}
	}

	return ui.Book{Title: book.Title, Author: book.Author, Chapters: chapters, TOC: toc}, nil
}

func (b *Bridge) Layout(spine int, page m.Rect, style ui.Style) (ui.Layout, error) {
	inset := m.Rect{
		X: round(page.W * insetX),
		Y: round(page.H * insetY),
		W: round(page.W * insetX),
		H: round(page.H * insetY),
	}
	viewport := engine.Viewport{
		Width:  page.W,
		Height: page.H,
		Top:    inset.Y,
		Right:  inset.W,
		Bottom: inset.H,
		Left:   inset.X,
	}

	pages, err := b.e.Layout(spine, viewport, toStyle(style))
	if err != nil {
		return ui.Layout{}, err
	}

	out := ui.Layout{Spine: spine, Pages: make([]ui.PageMeta, len(pages))}
	for i, p := range pages {
		out.Pages[i] = ui.PageMeta{
			Box:      page,
			Inset:    inset,
			LineFrom: p.LineFrom,
			LineLen:  p.LineLen,
			Progress: p.Progress,
		}
	}
	return out, nil
}

func (b *Bridge) Bitmap(page int, scale float32, style ui.Style) (ui.Raster, error) {
	bitmap, err := b.e.Bitmap(page, scale, toStyle(style))
	if err != nil {
		return ui.Raster{}, err
	}
	return ui.Raster{W: bitmap.W, H: bitmap.H, RGBA: bitmap.RGBA}, nil
}

func (b *Bridge) Hit(page int, x, y float32) (ui.Location, bool) {
	loc, ok := b.e.Hit(page, x, y)
	return ui.Location{Page: loc.Page, Line: loc.Line, Byte: loc.Byte}, ok
}

func (b *Bridge) LinkAt(page int, x, y float32) (string, bool) { return b.e.LinkAt(page, x, y) }

func (b *Bridge) Selection(a, z ui.Location) (ui.TextSelection, bool) {
	sel, ok := b.e.Selection(fromLocation(a), fromLocation(z))
	if !ok {
		return ui.TextSelection{}, false
	}
	rects := make([]m.Rect, len(sel.Rects))
	for i, r := range sel.Rects {
		rects[i] = m.Rect{X: r.X, Y: r.Y, W: r.W, H: r.H}
	}
	return ui.TextSelection{
		Start: ui.Location{Page: sel.Start.Page, Line: sel.Start.Line, Byte: sel.Start.Byte},
		End:   ui.Location{Page: sel.End.Page, Line: sel.End.Line, Byte: sel.End.Byte},
		Rects: rects,
		Text:  sel.Text,
	}, true
}

func (b *Bridge) Progress(loc ui.Location) (float32, bool) {
	return b.e.Progress(fromLocation(loc))
}

func (b *Bridge) FragmentLine(fragment string) (int, bool) { return b.e.FragmentLine(fragment) }

func (b *Bridge) ResolveLink(href string) (ui.EngineLink, bool) {
	link, ok := b.e.ResolveLink(href)
	if !ok {
		return ui.EngineLink{}, false
	}
	return ui.EngineLink{Spine: link.Spine, Fragment: link.Fragment, External: link.External}, true
}

func (b *Bridge) RegisterFont(data []byte) (int, error) { return b.e.RegisterFont(data) }

func (b *Bridge) BaseFontSize() float32 { return b.e.BaseFontSize() }

func (b *Bridge) TOC() []ui.TOCEntry {
	toc := b.e.TOC()
	if len(toc) == 0 {
		return nil
	}
	out := make([]ui.TOCEntry, len(toc))
	for i, e := range toc {
		out[i] = ui.TOCEntry{Label: e.Label, Spine: e.Spine, Fragment: e.Fragment, Depth: e.Depth}
	}
	return out
}

func (b *Bridge) Close() error { return b.e.Close() }

func toStyle(s ui.Style) engine.Style {
	return engine.Style{
		FontScale:     s.FontScale,
		LineScale:     s.LineScale,
		LetterSpacing: s.LetterSpacing,
		Align:         s.Align,
		Ink:           s.Ink,
		Paper:         s.Paper,
		Bold:          s.Bold,
	}
}

func fromLocation(l ui.Location) engine.Location {
	return engine.Location{Page: l.Page, Line: l.Line, Byte: l.Byte}
}

// round 把内边距取整到 0.5 点，避免 1x 屏上出现半像素描边。
func round(v float32) float32 {
	return float32(int(v*2+0.5)) / 2
}
