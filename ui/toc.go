package ui

import (
	"fmt"

	. "github.com/egoist/mygo/ui"
)

// tocEntries 返回目录；书没有单独目录时退化成章节列表。
func (a *App) tocEntries() []TOCEntry {
	if len(a.toc) > 0 {
		return a.toc
	}
	out := make([]TOCEntry, 0, len(a.book.Chapters))
	for _, ch := range a.book.Chapters {
		out = append(out, TOCEntry{Label: ch.Title, Spine: ch.Spine})
	}
	return out
}

func tocLabel(e TOCEntry, i int) string {
	if e.Label != "" {
		return e.Label
	}
	if e.Spine >= 0 {
		return fmt.Sprintf("第 %d 章", e.Spine+1)
	}
	return fmt.Sprintf("目录项 %d", i+1)
}

// tocRow 是目录里的一行，手机整屏和桌面侧栏共用。
func (a *App) tocRow(c *Context, e TOCEntry, i int) {
	label := tocLabel(e, i)
	btn := Button(c.Key([2]int{a.spine, i}), label).FillWidth().TextAlign(Start)
	if e.Depth > 0 {
		btn.PaddingStart(10 + 14*float32(e.Depth))
	}
	btn.Disabled(e.Spine < 0).OnClick(func() { a.goTOC(e) })
}

// buildTOCScreen 是 compact 的全屏目录，左上角“返回”。
func (a *App) buildTOCScreen(c *Context) {
	t := c.Theme()
	Column(c).Fill().Background(t.Background).Children(func() {
		Row(c).Height(56).PaddingX(8).Gap(8).AlignItems(Center).Children(func() {
			Button(c.Key("toc-back"), "返回").OnClick(func() { a.screen = screenReader })
			Text(c, "目录").FontSize(18).Bold()
		})
		entries := a.tocEntries()
		Scroll(c).Grow(1).Padding(8).Children(func() {
			Column(c).Gap(2).Children(func() {
				for i, e := range entries {
					a.tocRow(c, e, i)
				}
			})
		})
	})
}

// tocSidebar 是 regular 的左侧目录栏。
func (a *App) tocSidebar(c *Context) {
	t := c.Theme()
	Column(c).Width(sidebarWidth).Background(t.Surface).Children(func() {
		Row(c).Height(regularBarHeight).PaddingX(12).Gap(6).AlignItems(Center).Children(func() {
			Text(c, "目录").Bold().Grow(1)
			Button(c.Key("sidebar-close"), "收起").OnClick(func() { a.sidebar = false })
		})
		entryScroll := Scroll(c).Grow(1).Padding(8)
		entryScroll.Children(func() {
			Column(c).Gap(2).Children(func() {
				for i, e := range a.tocEntries() {
					a.tocRow(c, e, i)
				}
			})
		})
	})
}
