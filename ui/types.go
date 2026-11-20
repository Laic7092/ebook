// TODO: 换成 internal/engine
//
// 这个文件是 docs/engine-api.md 的本地副本：引擎包出现之前，UI 只依赖这里的
// Engine 接口，方便在没有引擎实现时编译和测试。将来 internal/engine 落地后，
// 把 ui.Engine 换成 engine.Engine 即可，视图代码不需要改。
//
// 与引擎接口一一对应的命名差异只有两处，因为 ui 包本身已经有同名类型：
//   - Bitmap    -> Raster（引擎的位图结果）
//   - Selection -> TextSelection（引擎的选区）
//
// 其余类型名与字段和 docs/engine-api.md 保持一致。
package ui

import m "github.com/egoist/mygo/ui"

// Book 是一本书的书目信息与目录。
type Book struct {
	Title    string
	Author   string
	Chapters []Chapter
	TOC      []TOCEntry
}

// Chapter 是阅读顺序里的一章。
type Chapter struct {
	Spine int
	Title string
}

// TOCEntry 是目录里的一项。Spine 为 -1 表示不在阅读顺序里（封面、注释页）。
type TOCEntry struct {
	Label    string
	Spine    int
	Fragment string
	Depth    int
}

// PageMeta 是一页的几何与行范围。单位是点（1x）。
type PageMeta struct {
	Box      m.Rect  // 页面盒子
	Inset    m.Rect  // 四边内边距
	LineFrom int     // 本页第一行
	LineLen  int     // 本页行数
	Progress float32 // 本章进度 0..1
}

// Layout 是一章的排版结果。
type Layout struct {
	Spine int
	Pages []PageMeta
}

// Raster 是光栅化后的页面：W*H*4 字节，行优先，非预乘 alpha。
type Raster struct {
	W, H int
	RGBA []byte
}

// Location 是书内位置。
type Location struct{ Page, Line, Byte uint32 }

// TextSelection 是选区及它在页内的矩形。
type TextSelection struct {
	Start, End Location
	Rects      []m.Rect
	Text       string
}

// EngineLink 是页面里链接的目标（mygo ui 已有 Link 函数，故加前缀）。
type EngineLink struct {
	Spine    int
	Fragment string
	External bool
}

// Style 是宿主侧的阅读设置。
type Style struct {
	FontScale     float32 // 只缩放文档字号/行高，不改变页面几何
	LineScale     float32 // 行距倍数，0 表示 1
	LetterSpacing float32 // 额外的字距（px）
	Align         int     // 0=保留原书 1=左 2=中 3=右 4=两端
	Ink           uint32  // 0xRRGGBBAA，0 表示不做墨色/伽马修正
	Paper         uint32  // 0xRRGGBBAA，0 表示白纸
	Bold          bool
}

// Engine 是 UI 依赖的最小引擎契约，方法签名见 docs/engine-api.md。
//
// 引擎方法不是并发安全的。App 用一把锁串行化所有调用：排版与光栅化都在
// 后台 job 里做，结果通过 Window.Update 回投到 UI 线程。
type Engine interface {
	// Open 打开一本 EPUB 或 TXT（按魔数识别）。
	Open(name string, data []byte) (Book, error)

	// Layout 排版一章：viewport 是单页可用区域（点），style 是阅读设置。
	Layout(spine int, viewport m.Rect, style Style) (Layout, error)

	// Bitmap 光栅化一页。scale 是设备像素比（1..3）。
	Bitmap(page int, scale float32, style Style) (Raster, error)

	Hit(page int, x, y float32) (Location, bool)
	LinkAt(page int, x, y float32) (string, bool)
	Selection(a, b Location) (TextSelection, bool)
	Progress(loc Location) (float32, bool)
	FragmentLine(fragment string) (int, bool)
	ResolveLink(href string) (EngineLink, bool)

	// RegisterFont 注册一个字体（sfnt/woff），返回字体 id。
	RegisterFont(data []byte) (int, error)
	BaseFontSize() float32
	TOC() []TOCEntry

	Close() error
}
