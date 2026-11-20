package ui

import (
	"fmt"

	myui "github.com/egoist/mygo/ui"
)

// settingsSheet 是阅读设置：字号、行距、主题、加粗。
//
// 全是"改 State → 请求一帧"的按钮/开关：`ensureRender` 每帧比较
// `Style`，设置一变下一帧就重新排版并光栅化。
func (a *App) settingsSheet(c *myui.Context, th *myui.Theme) {
	myui.Column(c).Absolute().Top(0).Left(0).Right(0).Bottom(0).
		Background(th.Background).Children(func() {
		myui.Row(c).Height(56).Padding(0, 8, 0, 16).Gap(8).
			AlignItems(myui.Center).Children(func() {
			if myui.Button(c, "←").Label("关闭设置").Height(44).PaddingX(14).Clicked() {
				a.settingsOpen = false
				a.requestFrame()
			}
			myui.Text(c, "阅读设置").FontSize(17).Bold().Grow(1)
		})
		myui.Divider(c)

		myui.Scroll(c).Fill().Children(func() {
			myui.Column(c).Padding(16, 16, 24, 16).Gap(16).Children(func() {
				a.stepperRow(c, th, "字号", fmt.Sprintf("%d%%", percent(a.State.FontScale)),
					func() { a.adjustFont(-0.1) },
					func() { a.adjustFont(0.1) },
				)
				a.stepperRow(c, th, "行距", fmt.Sprintf("%.1f×", a.State.LineScale),
					func() { a.adjustLine(-0.1) },
					func() { a.adjustLine(0.1) },
				)

				myui.Row(c).AlignItems(myui.Center).Gap(12).Children(func() {
					myui.Text(c, "主题").Width(64).TextColor(th.TextMuted)
					paper := myui.Button(c, "纸色").Height(40).PaddingX(16)
					night := myui.Button(c, "夜间").Height(40).PaddingX(16)
					if a.State.Theme == ThemePaper {
						paper.Bold()
					} else {
						night.Bold()
					}
					if paper.Clicked() && a.State.Theme != ThemePaper {
						a.State.Theme = ThemePaper
						a.requestFrame()
					}
					if night.Clicked() && a.State.Theme != ThemeNight {
						a.State.Theme = ThemeNight
						a.requestFrame()
					}
				})

				myui.Row(c).AlignItems(myui.Center).Gap(12).Children(func() {
					myui.Text(c, "加粗").Width(64).TextColor(th.TextMuted)
					if myui.Switch(c, &a.State.Bold).Changed() {
						a.requestFrame()
					}
					myui.Text(c, "让正文笔画更重（不改排版位置）").FontSize(13).
						TextColor(th.TextMuted)
				})

				myui.Text(c, "设置会立刻重新排版；关闭窗口后会记住。").FontSize(13).
					TextColor(th.TextMuted)
			})
		})
	})
}

// stepperRow 是一行"标签 + − + 值 + +"。
func (a *App) stepperRow(
	c *myui.Context,
	th *myui.Theme,
	label string,
	value string,
	minus func(),
	plus func(),
) {
	myui.Row(c).AlignItems(myui.Center).Gap(12).Children(func() {
		myui.Text(c, label).Width(64).TextColor(th.TextMuted)
		if myui.Button(c, "−").Label("减小" + label).Height(40).Width(40).Clicked() {
			minus()
		}
		myui.Text(c, value).Width(72).TextAlign(myui.Center)
		if myui.Button(c, "+").Label("增大" + label).Height(40).Width(40).Clicked() {
			plus()
		}
	})
}

// adjustFont 按 10% 步进改字号。
func (a *App) adjustFont(delta float64) {
	a.State.FontScale = clampF(a.State.FontScale+delta, 0.6, 2.4)
	a.requestFrame()
}

// adjustLine 按 0.1 步进改行距。
func (a *App) adjustLine(delta float64) {
	a.State.LineScale = clampF(a.State.LineScale+delta, 0.8, 2.0)
	a.requestFrame()
}

func clampF(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func percent(v float64) int {
	return int(v*100 + 0.5)
}
