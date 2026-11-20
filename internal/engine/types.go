// Package engine 是阅读器引擎的 Go 侧门面。
//
// 实现在 iOS/Android/桌面上走 cgo 链接 Rust 静态库（`ebook-abi`）；
// 没有 cgo 的构建会退化成只返回 ErrUnsupported 的桩，好让 UI 仍然能编译。
//
// 一个 Engine 一本会话：没有全局状态，可以同时打开多本（当前书 + 预打开）。
// 方法不是并发安全的，调用方自己串行化（UI 都在一个 goroutine 里）。
package engine

import "errors"

// ErrUnsupported 表示这次构建没有链接引擎（没有 cgo，或平台没有预编译库）。
var ErrUnsupported = errors.New("engine: 这个平台还没有构建引擎静态库")

type Book struct {
	Title    string
	Author   string
	Language string
	Cover    string
	Chapters []Chapter
	TOC      []TOCEntry
	Fonts    []FontFace
}

type Chapter struct {
	Spine  int
	ID     string
	Href   string
	Linear bool
}

type TOCEntry struct {
	Label    string
	Spine    int // -1 表示不在阅读顺序里（封面、注释页）
	Fragment string
	Depth    int
}

type FontFace struct {
	Family string
	Weight int
	Href   string
}

// Rect 是页面几何，单位是点（1x），与 IR 一致。
type Rect struct{ X, Y, W, H float32 }

// PageMeta 是内容盒（视口减内边距）与行区间、本章进度。
type PageMeta struct {
	Width    float32
	Height   float32
	Top      float32
	Right    float32
	Bottom   float32
	Left     float32
	LineFrom int
	LineLen  int
	Progress float32
}

// Bitmap.RGBA 是 W*H*4 字节，行优先，非预乘 alpha。
type Bitmap struct {
	W, H int
	RGBA []byte
}

type Location struct{ Page, Line, Byte uint32 }

type SelectionRect struct {
	Page uint32
	X, Y float32
	W, H float32
}

type Selection struct {
	Start, End Location
	Rects      []SelectionRect
	Text       string
}

type Link struct {
	Spine    int
	Fragment string
	External bool
}

// Style 是宿主侧的阅读设置。
type Style struct {
	FontScale     float32
	LineScale     float32
	LetterSpacing float32
	Align         int // 0=保留原书 1=左 2=中 3=右 4=两端
	Ink           uint32
	Paper         uint32
	Bold          bool
}

// Viewport 是排版视口：页面区域（点）+ 四边内边距。
type Viewport struct {
	Width, Height            float32
	Top, Right, Bottom, Left float32
}

type Engine interface {
	// Open 打开一本 EPUB 或 TXT（按魔数识别）。
	Open(name string, data []byte) (Book, error)
	Layout(spine int, viewport Viewport, style Style) ([]PageMeta, error)
	Bitmap(page int, scale float32, style Style) (Bitmap, error)
	Hit(page int, x, y float32) (Location, bool)
	LinkAt(page int, x, y float32) (string, bool)
	Selection(a, b Location) (Selection, bool)
	Progress(loc Location) (float32, bool)
	FragmentLine(fragment string) (int, bool)
	ResolveLink(href string) (Link, bool)
	RegisterFont(data []byte) (int, error)
	RegisterFontFace(index int) (int, error)
	FontFaces() int
	BaseFontSize() float32
	TOC() []TOCEntry
	Pages() []PageMeta
	Close() error
}
