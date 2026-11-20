package ui

import . "github.com/egoist/mygo/ui"

// settingsControls 是字号、行距、主题、加粗。两种布局共用这一份。
func (a *App) settingsControls(c *Context, compact bool) {
	t := c.Theme()
	Column(c).Gap(18).Children(func() {
		// 字号
		Row(c).Gap(8).AlignItems(Center).Children(func() {
			Text(c, "字号").Grow(1)
			if Button(c.Key("font-dec"), "字号 -").Clicked() {
				a.adjustFont(-0.1)
			}
			Textf(c, "%.0f pt", a.fontPoints()).Width(64).TextAlign(Center).TextColor(t.TextMuted)
			if Button(c.Key("font-inc"), "字号 +").Clicked() {
				a.adjustFont(0.1)
			}
		})

		// 行距
		Row(c).Gap(8).AlignItems(Center).Children(func() {
			Text(c, "行距").Grow(1)
			if Button(c.Key("line-dec"), "行距 -").Clicked() {
				a.adjustLine(-0.1)
			}
			Textf(c, "%.1f×", a.style.LineScale).Width(52).TextAlign(Center).TextColor(t.TextMuted)
			if Button(c.Key("line-inc"), "行距 +").Clicked() {
				a.adjustLine(0.1)
			}
		})

		// 主题纸色 / 夜间
		Column(c).Gap(6).Children(func() {
			Text(c, "主题").TextColor(t.TextMuted).FontSize(12)
			Row(c).Gap(8).Children(func() {
				for i, name := range []string{"纸白", "米黄", "夜间"} {
					if Button(c.Key([2]int{1, i}), name).Grow(1).Clicked() {
						a.setTheme(readingTheme(i))
					}
				}
			})
		})

		// 加粗
		Row(c).Gap(8).AlignItems(Center).Children(func() {
			Text(c, "加粗").Grow(1)
			Switch(c.Key("bold"), &a.style.Bold).Label("加粗正文")
		})

		if compact {
			Text(c, "排版完成后自动回到当前阅读位置。").FontSize(12).TextColor(t.TextMuted)
		}
	})
}

// buildSettingsScreen 是 compact 的全屏设置面板。
func (a *App) buildSettingsScreen(c *Context) {
	t := c.Theme()
	Column(c).Fill().Background(t.Background).Children(func() {
		Row(c).Height(56).PaddingX(8).Gap(8).AlignItems(Center).Children(func() {
			Button(c.Key("settings-back"), "返回").OnClick(func() { a.screen = screenReader })
			Text(c, "阅读设置").FontSize(18).Bold()
		})
		Scroll(c).Grow(1).Padding(16).Children(func() {
			a.settingsControls(c, true)
		})
	})
}

// settingsPanel 是 regular 右侧的设置浮层。
func (a *App) settingsPanel(c *Context) {
	t := c.Theme()
	Column(c).Absolute().Top(regularBarHeight).Bottom(regularFootHeight).Right(0).Width(320).
		Padding(16).Gap(16).Background(t.Surface).Border(1, t.Border).Children(func() {
		Row(c).Gap(8).AlignItems(Center).Children(func() {
			Text(c, "阅读设置").FontSize(16).Bold().Grow(1)
			Button(c.Key("settings-close"), "关闭").OnClick(func() { a.screen = screenReader })
		})
		Scroll(c).Grow(1).Children(func() {
			a.settingsControls(c, false)
		})
	})
}
