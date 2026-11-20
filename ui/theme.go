package ui

import . "github.com/egoist/mygo/ui"

// readingTheme 是阅读区主题，和系统的明暗外观解耦：读者可以选纸白、
// 米黄或夜间，而不改系统设置。
type readingTheme int

const (
	ThemePaper readingTheme = iota // 纸白
	ThemeSepia                     // 米黄
	ThemeNight                     // 夜间
)

func (t readingTheme) String() string {
	switch t {
	case ThemeSepia:
		return "米黄"
	case ThemeNight:
		return "夜间"
	default:
		return "纸白"
	}
}

// palette 是某一档主题在窗口和引擎两边要用的颜色。
// Paper/Ink 是引擎的 0xRRGGBBAA；Ink 为 0 表示保留引擎默认墨色。
type palette struct {
	dark     bool
	bg       Color
	surface  Color
	surfaceH Color
	press    Color
	border   Color
	text     Color
	muted    Color
	paper    uint32
	ink      uint32
}

func paletteFor(t readingTheme) palette {
	switch t {
	case ThemeSepia:
		return palette{
			bg: Hex("#f4ecd8"), surface: Hex("#ece2cb"), surfaceH: Hex("#e3d7bc"),
			press: Hex("#d9cbaa"), border: Hex("#d8c9aa"),
			text: Hex("#40382c"), muted: Hex("#7d7361"),
			paper: 0xf4ecd8ff,
		}
	case ThemeNight:
		return palette{
			dark: true,
			bg:   Hex("#14161a"), surface: Hex("#1e2229"), surfaceH: Hex("#272c35"),
			press: Hex("#303641"), border: Hex("#333944"),
			text: Hex("#c8ccd3"), muted: Hex("#8b93a1"),
			paper: 0x14161aff, ink: 0xc8ccd3ff,
		}
	default:
		return palette{
			bg: Hex("#fbfaf7"), surface: Hex("#f1efe9"), surfaceH: Hex("#e8e5dd"),
			press: Hex("#dfdbd1"), border: Hex("#dcd8ce"),
			text: Hex("#20201e"), muted: Hex("#74716a"),
			paper: 0xfbfaf7ff,
		}
	}
}

// applyPalette 把主题里和引擎有关的两项写进 Style。
func (a *App) applyPalette() {
	p := paletteFor(a.theme)
	a.style.Paper = p.paper
	a.style.Ink = p.ink
}

// applyTheme 每帧设置窗口主题。系统明暗不参与：阅读区主题由读者选择。
func (a *App) applyTheme(c *Context) {
	p := paletteFor(a.theme)
	t := LightTheme()
	if p.dark {
		t = DarkTheme()
	}
	t.Background = p.bg
	t.Surface = p.surface
	t.SurfaceHover = p.surfaceH
	t.SurfacePressed = p.press
	t.Border = p.border
	t.Text = p.text
	t.TextMuted = p.muted
	c.SetTheme(t)
}
