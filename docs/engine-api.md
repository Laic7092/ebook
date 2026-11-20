# 引擎接口（UI 侧的契约）

UI 只认这个接口，不关心它是 Rust 静态库、wasm 还是测试用的假实现。
实现由引擎侧负责（`internal/engine`），**UI 不要修改这个包**。

```go
package engine

type Book struct {
	Title    string
	Author   string
	Chapters []Chapter
	TOC      []TOCEntry
}

type Chapter struct {
	Spine int
	Title string
}

type TOCEntry struct {
	Label    string
	Spine    int    // -1 表示不在阅读顺序里（封面、注释页）
	Fragment string // 可空
	Depth    int
}

// Rect 是页面几何，单位是"点"（1x），与 IR 一致；设备像素缩放由 Bitmap 的 scale 负责。
type Rect struct{ X, Y, W, H float32 }

type PageMeta struct {
	Box      Rect    // 页面盒子
	Inset    Rect    // 四边内边距（left/right/top/bottom 落在 X/W/Y/H 里）
	LineFrom int     // 本页第一行
	LineLen  int     // 本页行数
	Progress float32 // 本章进度 0..1
}

type Layout struct {
	Spine int
	Pages []PageMeta
}

// Bitmap.RGBA 是 w*h*4 字节，行优先，非预乘 alpha。
type Bitmap struct {
	W, H int
	RGBA []byte
}

type Location struct{ Page, Line, Byte uint32 }

type Selection struct {
	Start, End Location
	Rects      []Rect // 页内坐标
	Text       string
}

type Link struct {
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

type Engine interface {
	// Open 打开一本 EPUB 或 TXT（按魔数识别）。返回书目信息与目录。
	Open(name string, data []byte) (Book, error)

	// Layout 排版一章：viewport 是可用页面区域（点），style 是阅读设置。
	Layout(spine int, viewport Rect, style Style) (Layout, error)

	// Bitmap 光栅化一页。scale 是设备像素比（1..3）。
	Bitmap(page int, scale float32, style Style) (Bitmap, error)

	Hit(page int, x, y float32) (Location, bool)
	LinkAt(page int, x, y float32) (string, bool)
	Selection(a, b Location) (Selection, bool)
	Progress(loc Location) (float32, bool)
	FragmentLine(fragment string) (int, bool)
	ResolveLink(href string) (Link, bool)

	// RegisterFont 注册一个字体（sfnt/woff），返回字体 id。
	RegisterFont(data []byte) (int, error)
	BaseFontSize() float32
	TOC() []TOCEntry

	Close() error
}
```

## 约定

- **无全局状态**：一个 `Engine` 一本会话，UI 可以同时持有多个（当前书 + 预打开）。
- `Layout` 只对**当前章**有效；换章要重新 `Layout`。`Layout` 是幂等的，同样的输入给同样的页。
- `Bitmap` 的坐标与 `Layout` 一致：1x 点坐标 × scale = 位图像素。
- `Hit` / `LinkAt` 的坐标是**页内点坐标**（不是设备像素）。
- 引擎层不做 I/O、不读时钟；字体与书籍字节都由宿主传入。
- 并发：`Engine` 的方法不是并发安全的，UI 用一把锁或单 goroutine（`Window.Update` 回调）访问。
