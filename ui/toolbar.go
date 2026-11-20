package ui

import . "github.com/egoist/mygo/ui"

// compactTopBar 是页面上方半透明条：目录、书名、设置。
func (a *App) compactTopBar(c *Context) {
	p := paletteFor(a.theme)
	Row(c).Absolute().Top(0).FillWidth().Height(compactTopHeight).
		PaddingY(8).PaddingX(10).Gap(8).AlignItems(Center).
		Background(RGBA(p.bg.R, p.bg.G, p.bg.B, 0.93)).
		Children(func() {
			Button(c.Key("compact-toc"), "目录").OnClick(a.openTOC)
			Text(c, a.bookTitle()).Grow(1).TextColor(p.text).TextAlign(Center).MaxWidthPercent(60)
			Button(c.Key("compact-settings"), "设置").OnClick(a.openSettings)
		})
}

// compactFootBar 是页面下方半透明条：进度和页码。
func (a *App) compactFootBar(c *Context) {
	p := paletteFor(a.theme)
	Row(c).Absolute().Bottom(0).FillWidth().Height(compactFootHeight).
		PaddingX(14).Gap(10).AlignItems(Center).
		Background(RGBA(p.bg.R, p.bg.G, p.bg.B, 0.93)).
		Children(func() {
			Progress(c, a.progress()).Grow(1).Label("阅读进度")
			Text(c, a.pageLabel()).FontSize(12).TextColor(p.muted)
		})
}

// regularTopBar 是一行细工具栏：目录、书名、字号、主题、设置。
func (a *App) regularTopBar(c *Context) {
	t := c.Theme()
	Row(c).Height(regularBarHeight).PaddingX(12).Background(t.Background).Children(func() {
		Toolbar(c, func() {
			Button(c.Key("sidebar-toggle"), "目录").OnClick(a.openTOC)
			Text(c, a.bookTitle()).Bold().Grow(1).MaxWidthPercent(50)
			Spacer(c)
			Button(c.Key("font-smaller"), "A-").OnClick(func() { a.adjustFont(-0.1) })
			Textf(c, "%.0f", a.fontPoints()).FontSize(12).TextColor(t.TextMuted).Width(28).TextAlign(Center)
			Button(c.Key("font-larger"), "A+").OnClick(func() { a.adjustFont(0.1) })
			Button(c.Key("cycle-theme"), a.theme.String()).OnClick(func() {
				a.setTheme((a.theme + 1) % 3)
			})
			Button(c.Key("open-settings"), "设置").OnClick(a.openSettings)
		}).Label("阅读工具栏")
	})
}

// regularFootBar 是底部进度条和页码。
func (a *App) regularFootBar(c *Context) {
	t := c.Theme()
	Row(c).Height(regularFootHeight).PaddingX(16).Gap(12).AlignItems(Center).Children(func() {
		Progress(c, a.progress()).Grow(1).Label("阅读进度")
		Text(c, a.pageLabel()).FontSize(12).TextColor(t.TextMuted)
	})
}
