package ui

import (
	"fmt"

	myui "github.com/egoist/mygo/ui"
)

// openTOC 切换目录。手机上是整屏，其余情况同一套浮层。
func (a *App) openTOC() {
	a.tocOpen = !a.tocOpen
	a.settingsOpen = false
	a.requestFrame()
}

// tocDrawer 是目录浮层：标题栏 + 可滚动条目。点条目跳到那一章，
// 如果条目带 fragment，排版完成后由 `FragmentLine` 换算成页。
func (a *App) tocDrawer(c *myui.Context, th *myui.Theme) {
	myui.Column(c).Absolute().Top(0).Left(0).Right(0).Bottom(0).
		Background(th.Background).Children(func() {
		myui.Row(c).Height(56).Padding(0, 8, 0, 16).Gap(8).
			AlignItems(myui.Center).Children(func() {
			if myui.Button(c, "←").Label("关闭目录").Height(44).PaddingX(14).Clicked() {
				a.tocOpen = false
				a.requestFrame()
			}
			myui.Text(c, "目录").FontSize(17).Bold().Grow(1)
		})
		myui.Divider(c)

		if len(a.book.TOC) == 0 {
			myui.Text(c, "这本书没有目录").TextColor(th.TextMuted).Padding(16)
			return
		}
		myui.Scroll(c).Fill().Children(func() {
			myui.Column(c).Padding(0, 8, 16, 8).Gap(2).Children(func() {
				for _, entry := range a.book.TOC {
					a.tocItem(c, th, entry)
				}
			})
		})
	})
}

// tocItem 是目录里的一行。缩进跟着 `depth`，当前章的条目加粗。
func (a *App) tocItem(c *myui.Context, th *myui.Theme, entry TOCEntry) {
	label := entry.Label
	if label == "" {
		label = "（无标题）"
	}
	indent := float32(12 + int(entry.Depth)*14)
	item := myui.Button(c, label).Height(44).PaddingX(indent).FillWidth().
		Label(fmt.Sprintf("跳到：%s", label))
	if entry.Spine == a.State.Spine {
		item.Bold().Background(th.SurfaceHover)
	}
	if entry.Spine < 0 {
		item.Disabled(true) // 封面、注释页不在阅读顺序里
		return
	}
	if item.Clicked() {
		a.jumpTo(entry)
	}
}

// jumpTo 跳到目录项。fragment 要等目标章排版完成后才能换算成行，
// 所以只记下来，交给 render 流程处理。
func (a *App) jumpTo(entry TOCEntry) {
	a.State.Spine = entry.Spine
	a.State.Page = 0
	a.pendingFragment = entry.Fragment
	a.pendingLast = false
	a.layoutValid = false
	a.tocOpen = false
	a.requestFrame()
}
