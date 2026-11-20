// 引擎类型的别名。
//
// UI 只依赖 internal/engine 的契约（docs/engine-api.md）；这里把名字收进
// package ui，省得每个文件都写 `engine.`，也方便测试注入假实现。
package ui

import (
	"github.com/Laic7092/ebook/internal/engine"
	myui "github.com/egoist/mygo/ui"
)

type (
	Engine    = engine.Engine
	Book      = engine.Book
	Chapter   = engine.Chapter
	TOCEntry  = engine.TOCEntry
	FontFace  = engine.FontFace
	PageMeta  = engine.PageMeta
	Style     = engine.Style
	Viewport  = engine.Viewport
	Rect      = engine.Rect
	Location  = engine.Location
	Selection = engine.Selection
	Link      = engine.Link

	// 引擎的 Bitmap 是 RGBA 字节，MyGo 的 Bitmap 是 GPU 纹理，别混。
	Bitmap = engine.Bitmap

	Context    = myui.Context
	Element    = myui.Element
	ListState  = myui.ListState
	InputEvent = myui.InputEvent
)

// Layout 是一章的排版结果：页号即下标。
type Layout struct {
	Spine int
	Pages []PageMeta
}
