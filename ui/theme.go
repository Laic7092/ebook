package ui

import myui "github.com/egoist/mygo/ui"

// uiTheme 返回阅读界面的主题：纸色或夜间。控件（按钮、滑杆、列表选中）
// 跟着一起变，不只是换背景。
func (a *App) uiTheme() *myui.Theme {
	base := myui.LightTheme()
	if a.State.Theme == ThemeNight {
		base = myui.DarkTheme()
	}
	th := *base
	t := &th

	t.Dark = a.State.Theme == ThemeNight
	if t.Dark {
		t.Background = myui.Hex("#12110F")
		t.Surface = myui.Hex("#1D1B18")
		t.SurfaceHover = myui.Hex("#26231F")
		t.SurfacePressed = myui.Hex("#2E2A25")
		t.Border = myui.Hex("#3A352E")
		t.Text = myui.Hex("#E8E2D6")
		t.TextMuted = myui.Hex("#9B9384")
		t.Accent = myui.Hex("#C79A5B")
		t.AccentHover = myui.Hex("#D7AC70")
		t.AccentPressed = myui.Hex("#B2864A")
		t.AccentText = myui.Hex("#1A1208")
		t.Selection = myui.Hex("#3A2F22")
		t.Inverse = myui.Hex("#E8E2D6")
		t.InverseText = myui.Hex("#12110F")
	} else {
		t.Background = myui.Hex("#F6F1E7")
		t.Surface = myui.Hex("#FFFBF2")
		t.SurfaceHover = myui.Hex("#EFE7D8")
		t.SurfacePressed = myui.Hex("#E6DCC9")
		t.Border = myui.Hex("#D8CDB8")
		t.Text = myui.Hex("#2B2A26")
		t.TextMuted = myui.Hex("#7A7365")
		t.Accent = myui.Hex("#8A5A2B")
		t.AccentHover = myui.Hex("#9A6838")
		t.AccentPressed = myui.Hex("#74491F")
		t.AccentText = myui.Hex("#FFFBF2")
		t.Selection = myui.Hex("#D9C3A5")
		t.Inverse = myui.Hex("#2B2A26")
		t.InverseText = myui.Hex("#F6F1E7")
	}
	t.Focus = t.Accent
	t.FocusRingWidth = 2
	t.FocusRingOffset = 2
	// 阅读窗口即使不在前台也保留强调色，避免整页发灰。
	t.InactiveAccent = t.Accent
	t.Radius = 10
	return t
}
