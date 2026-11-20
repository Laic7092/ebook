package ui

import (
	"math"
	"strconv"

	myui "github.com/egoist/mygo/ui"
)

// View 画阅读界面。它每帧从 App 的纯数据重建：整页位图、工具栏、
// 底部进度、目录抽屉和设置面板。没有 Element 被存进结构体。
func (a *App) View(c *myui.Context) {
	a.normalize()
	th := a.uiTheme()
	c.SetTheme(th)

	w, h := c.Size()
	viewport := a.pageViewport(w, h)
	a.ensureRender(c, viewport)

	if !c.FocusTakesText() {
		c.OnWindowShortcut(0, myui.KeyRight, func() { a.nextPage() })
		c.OnWindowShortcut(0, myui.KeyPageDown, func() { a.nextPage() })
		c.OnWindowShortcut(0, myui.KeySpace, func() { a.nextPage() })
		c.OnWindowShortcut(0, myui.KeyLeft, func() { a.previousPage() })
		c.OnWindowShortcut(0, myui.KeyPageUp, func() { a.previousPage() })
		c.OnWindowShortcut(myui.Cmd, myui.KeyEqual, func() { a.adjustFont(0.1) })
		c.OnWindowShortcut(myui.Cmd, myui.KeyMinus, func() { a.adjustFont(-0.1) })
	}

	root := myui.Box(c).Fill().Background(th.Background)
	root.Children(func() {
		a.page(c, th, w, h)
		a.bottomBar(c, th)
		if a.chrome {
			a.topBar(c, th)
		}
		switch {
		case a.tocOpen:
			a.tocDrawer(c, th)
		case a.settingsOpen:
			a.settingsSheet(c, th)
		}
	})
}

// pageDrag 是页面上的按下/拖动状态，由 ui.Local 跟着页面元素保存。
type pageDrag struct {
	active         bool
	startX, startY float32
	dx, dy         float32
}

// page 是整页区域：位图按容器等比缩放、居中，点击三分区，水平滑动翻页。
func (a *App) page(c *myui.Context, th *myui.Theme, w, h float32) {
	top, right, bottom, left := a.pageInsets(w)
	page := myui.Box(c.Key("page")).Fill().Clip().
		Padding(top, right, bottom, left)

	// 手势状态留在页面元素上；窗口重建/切章不丢，元素消失时自然释放。
	drag := myui.Local(page, "drag", func() pageDrag { return pageDrag{} })
	page.HandleInput(func(ev myui.InputEvent) bool {
		switch ev.Kind {
		case myui.InputPointerDown:
			if ev.Button != 0 {
				return false
			}
			*drag = pageDrag{active: true, startX: ev.X, startY: ev.Y}
			return true
		case myui.InputPointerMove:
			if !drag.active {
				return false
			}
			drag.dx = ev.X - drag.startX
			drag.dy = ev.Y - drag.startY
			return true
		case myui.InputPointerUp:
			if !drag.active {
				return false
			}
			d := *drag
			*drag = pageDrag{}
			if ev.Canceled {
				return true
			}
			a.endGesture(w, ev.X, d.dx, d.dy)
			return true
		}
		return false
	})

	page.Children(func() {
		if a.bitmap != nil {
			img := myui.Image(c, a.bitmap).Fit(myui.Contain).
				ObjectPosition(0.5, 0.5).Fill().IgnoreInput(true)
			if drag.active && drag.dx != 0 {
				// 拖动时给一点跟手位移，阈值到了整页翻动。
				img.Translate(drag.dx*0.28, 0).Opacity(1 - 0.3*minf(1, absf(drag.dx)/maxf(1, w*0.6)))
			}
		} else {
			a.emptyPage(c, th)
		}
	})
}

// endGesture 区分水平滑动和三分区点击。
func (a *App) endGesture(width, x, dx, dy float32) {
	threshold := width * 0.14
	if threshold < 48 {
		threshold = 48
	}
	if threshold > 96 {
		threshold = 96
	}
	if absf(dx) >= threshold && absf(dx) > absf(dy)*1.3 {
		if dx < 0 {
			a.nextPage()
		} else {
			a.previousPage()
		}
		return
	}
	if absf(dx) > 10 || absf(dy) > 10 {
		return // 斜着拖了一下，不当作点击
	}
	switch {
	case x < width/3:
		a.previousPage()
	case x > width*2/3:
		a.nextPage()
	default:
		a.chrome = !a.chrome
		a.requestFrame()
	}
}

func (a *App) emptyPage(c *myui.Context, th *myui.Theme) {
	msg := "未打开书籍"
	sub := "把 EPUB 交给 ui.App.Open"
	if a.eng != nil && len(a.book.Chapters) > 0 {
		msg = "正在排版…"
		sub = "第 " + a.chapterTitle()
	}
	if a.lastErr != "" {
		msg = "排版失败"
		sub = a.lastErr
	}
	myui.Column(c).Fill().Center().Gap(8).IgnoreInput(true).Children(func() {
		myui.Text(c, msg).FontSize(18).Bold().TextColor(th.TextMuted)
		if sub != "" {
			myui.Text(c, sub).FontSize(13).TextColor(th.TextMuted).TextAlign(myui.Center)
		}
	})
}

// topBar 是书名和目录/设置按钮。工具栏隐藏时整块不建，页面拿到全部高度。
func (a *App) topBar(c *myui.Context, th *myui.Theme) {
	myui.Row(c).Absolute().Top(0).Left(0).Right(0).Height(56).
		Padding(0, 16, 0, 16).Gap(8).AlignItems(myui.Center).
		Background(th.Background).Children(func() {
		myui.Text(c, a.Title()).FontSize(17).Bold().SingleLine().Grow(1)
		toc := myui.Button(c, "目录").Height(44).PaddingX(14).Label("目录")
		if len(a.book.TOC) == 0 {
			toc.Disabled(true)
		}
		if toc.Clicked() {
			a.openTOC()
		}
		if myui.Button(c, "Aa").Height(44).PaddingX(14).Label("设置").Clicked() {
			a.settingsOpen = !a.settingsOpen
			a.tocOpen = false
			a.requestFrame()
		}
	})
}

// bottomBar 是本章进度和页码，永远显示；它不接收点击，点击穿透到页面。
func (a *App) bottomBar(c *myui.Context, th *myui.Theme) {
	progress := float32(0)
	total := a.pageCount()
	page := a.State.Page
	if total > 0 {
		if page >= total {
			page = total - 1
		}
		progress = a.layout.Pages[page].Progress
	} else {
		page = 0
	}
	if progress < 0 {
		progress = 0
	}
	if progress > 1 {
		progress = 1
	}

	myui.Column(c).Absolute().Bottom(0).Left(0).Right(0).
		Padding(0, 16, 10, 16).Gap(6).IgnoreInput(true).Children(func() {
		myui.Progress(c, float64(progress)).Height(4).Label("本章进度").
			TextColor(th.Accent).Background(th.Border)
		myui.Row(c).Justify(myui.SpaceBetween).AlignItems(myui.Center).Children(func() {
			myui.Text(c, a.chapterTitle()).FontSize(12).TextColor(th.TextMuted).
				SingleLine().Grow(1)
			label := "— / —"
			if total > 0 {
				label = strconv.Itoa(page+1) + " / " + strconv.Itoa(total)
			}
			myui.Text(c, label).FontSize(12).TextColor(th.TextMuted)
		})
	})
}

// nextPage 先翻本章；到章尾换下一章。
func (a *App) nextPage() {
	if n := a.pageCount(); n > 0 && a.State.Page+1 < n {
		a.State.Page++
		a.requestFrame()
		return
	}
	if a.nextChapter() {
		a.requestFrame()
	}
}

// previousPage 先翻本章；在章首换上一章并停在最后一页。
func (a *App) previousPage() {
	if a.State.Page > 0 {
		a.State.Page--
		a.requestFrame()
		return
	}
	if a.previousChapter() {
		a.pendingLast = true
		a.requestFrame()
	}
}

func (a *App) nextChapter() bool {
	chs := a.book.Chapters
	for i, ch := range chs {
		if ch.Spine != a.State.Spine {
			continue
		}
		if i+1 >= len(chs) {
			return false
		}
		a.State.Spine = chs[i+1].Spine
		a.State.Page = 0
		a.pendingFragment = ""
		a.pendingLast = false
		return true
	}
	if len(chs) == 0 {
		a.State.Spine++
		a.State.Page = 0
		return true
	}
	a.State.Spine = chs[0].Spine
	a.State.Page = 0
	return true
}

func (a *App) previousChapter() bool {
	chs := a.book.Chapters
	for i, ch := range chs {
		if ch.Spine != a.State.Spine {
			continue
		}
		if i == 0 {
			return false
		}
		a.State.Spine = chs[i-1].Spine
		a.State.Page = 0
		a.pendingFragment = ""
		a.pendingLast = true
		return true
	}
	return false
}

func (a *App) pageCount() int {
	if !a.layoutValid || a.layoutSpine != a.State.Spine {
		return 0
	}
	return len(a.layout.Pages)
}

func (a *App) chapterTitle() string {
	// spine 里没有标题（那是导航文档的职责），先看目录里有没有对应当前章的项。
	for _, entry := range a.book.TOC {
		if entry.Spine == a.State.Spine && entry.Label != "" {
			return entry.Label
		}
	}
	for i, ch := range a.book.Chapters {
		if ch.Spine == a.State.Spine {
			return "第 " + strconv.Itoa(i+1) + " 章"
		}
	}
	return a.book.Title
}

func minf(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}

func maxf(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

func absf(v float32) float32 { return float32(math.Abs(float64(v))) }
