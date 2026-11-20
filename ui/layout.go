package ui

import . "github.com/egoist/mygo/ui"

// 断点集中在这里。窗口宽度按内容区（已扣掉安全区）计算：
//   - < 720pt：compact，手机单页，工具栏浮在页面上；
//   - >= 720pt：regular，桌面；>= 1100pt 时双页对开。
const (
	compactMaxWidth   float32 = 720
	twoPageMinWidth   float32 = 1100
	compactMargin     float32 = 14
	regularMargin     float32 = 28
	sidebarWidth      float32 = 220
	regularBarHeight  float32 = 44
	regularFootHeight float32 = 36
	pageGap           float32 = 24
	compactTopHeight  float32 = 52
	compactFootHeight float32 = 34
)

type layoutMode int

const (
	layoutCompact layoutMode = iota
	layoutRegular
)

// pageBox 是一次布局里“页面区域”的几何。area 是页面可以画到的矩形，
// viewport 是交给引擎排版单页的区域，columns 是 1 或 2。
type pageBox struct {
	mode     layoutMode
	area     Rect
	viewport Rect
	columns  int
}

func breakpoint(w float32) layoutMode {
	if w < compactMaxWidth {
		return layoutCompact
	}
	return layoutRegular
}

// measurePages 用窗口可用尺寸算页面区域。视图按同样的数字搭 flex 容器，
// 保证 ui.Image 显示的区域和引擎排版用的 viewport 一致。
func measurePages(w, h float32, mode layoutMode, sidebar bool) pageBox {
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	if mode == layoutCompact {
		area := Rect{
			X: compactMargin,
			Y: compactMargin,
			W: maxF(1, w-2*compactMargin),
			H: maxF(1, h-2*compactMargin),
		}
		return pageBox{mode: mode, area: area, columns: 1,
			viewport: Rect{W: area.W, H: area.H}}
	}

	left := float32(0)
	if sidebar {
		left = sidebarWidth
	}
	area := Rect{
		X: left + regularMargin,
		Y: regularBarHeight + regularMargin,
		W: maxF(1, w-left-2*regularMargin),
		H: maxF(1, h-regularBarHeight-regularFootHeight-2*regularMargin),
	}
	cols := 1
	if w >= twoPageMinWidth {
		cols = 2
	}
	vw := area.W
	if cols == 2 {
		vw = (area.W - pageGap) / 2
	}
	return pageBox{
		mode: mode, area: area, columns: cols,
		viewport: Rect{W: maxF(1, vw), H: area.H},
	}
}

// buildCompact 是手机布局：单页铺满，工具栏和进度条浮在上下边缘。
func (a *App) buildCompact(c *Context, box pageBox) {
	switch a.screen {
	case screenTOC:
		a.buildTOCScreen(c)
		return
	case screenSettings:
		a.buildSettingsScreen(c)
		return
	}

	t := c.Theme()
	Box(c).Fill().Background(t.Background).Children(func() {
		// 页面与手势层。两者用同一个内边距，触摸坐标因此和页面区域一致。
		Box(c).Fill().Padding(compactMargin).Children(func() {
			Box(c).Fill().Children(func() {
				a.buildPageStack(c, box, 0)
			})
			// 空白 Box 在最上面接收按下/移动/抬起，自己判断热区和滑动。
			// 绝对定位让它覆盖页面，而不是在 Column 里再占一份高度。
			page := Box(c).Absolute().Fill()
			page.HandleInput(a.compactInput(box.area.W))
		})
		if a.toolbar {
			a.compactTopBar(c)
			a.compactFootBar(c)
		}
	})
}

// buildRegular 是桌面布局：左侧目录（可折叠）、顶部细工具栏、
// 中间单页或双页、底部进度条。
func (a *App) buildRegular(c *Context, box pageBox) {
	t := c.Theme()
	Box(c).Fill().Background(t.Background).Children(func() {
		Column(c).Fill().Children(func() {
			a.regularTopBar(c)
			Row(c).Grow(1).AlignItems(Stretch).Children(func() {
				if a.sidebar {
					a.tocSidebar(c)
				}
				Column(c).Grow(1).Children(func() {
					pages := Box(c).Grow(1).Padding(regularMargin)
					pages.Children(func() { a.buildPageStack(c, box, 0) })
					// 鼠标滚轮/触控板翻页。没有滚动容器时，滚轮直接到这里。
					pages.HandleInput(a.wheelInput())
					a.regularFootBar(c)
				})
			})
		})
		if a.screen == screenSettings {
			a.settingsPanel(c)
		}
	})
}

// buildPageStack 把页面放进可平移的容器。compact 拖动时跟手平移，
// regular 的偏移始终是 0。
func (a *App) buildPageStack(c *Context, box pageBox, _ float32) {
	e := Box(c).Fill()
	if off := a.swipeOffset(e); off != 0 {
		e.Translate(off, 0)
	}
	e.Children(func() { a.buildPages(c, box) })
}

func (a *App) bookTitle() string {
	if a.book.Title != "" {
		return a.book.Title
	}
	return "eBook"
}

func maxF(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}
