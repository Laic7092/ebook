package ui

import (
	"testing"
	"time"

	. "github.com/egoist/mygo/ui"
)

func newTestReader(t *testing.T) (*App, *fakeEngine, *Tester) {
	t.Helper()
	fe := newFakeEngine()
	a := NewReader(fe)
	a.Open("test.epub", nil)
	tt := NewTester(a.View, 390, 844)
	tt.Frame()
	if !a.hasLayout {
		t.Fatalf("first frame laid out nothing; texts %q", tt.Texts())
	}
	return a, fe, tt
}

// 1. compact 点击左右 1/3 翻页，中间切换工具栏。
func TestCompactTapZones(t *testing.T) {
	a, _, tt := newTestReader(t)
	if a.page != 0 {
		t.Fatalf("start page %d", a.page)
	}
	tt.ClickAt(370, 400)
	tt.Frame()
	if a.page != 1 {
		t.Fatalf("right third: page %d, want 1", a.page)
	}
	tt.ClickAt(370, 400)
	tt.Frame()
	if a.spine != 1 || a.page != 0 {
		t.Fatalf("past end: spine %d page %d, want 1/0", a.spine, a.page)
	}
	tt.ClickAt(20, 400)
	tt.Frame()
	if a.spine != 0 {
		t.Fatalf("left third at chapter start: spine %d, want 0", a.spine)
	}
}

// 2. 水平滑动超过 1/4 页宽翻页，未超过回弹。
func TestCompactSwipe(t *testing.T) {
	a, _, tt := newTestReader(t)
	tt.Press(340, 400)
	tt.Move(60, 400)
	tt.Release(60, 400)
	tt.Frame()
	if a.page != 1 {
		t.Fatalf("swipe left: page %d, want 1", a.page)
	}
	tt.Press(200, 400)
	tt.Move(170, 400) // 30pt，远小于 1/4 页宽
	tt.Release(170, 400)
	tt.Frame()
	if a.page != 1 {
		t.Fatalf("short drag turned page: %d", a.page)
	}
	if a.swipeSettling || a.swipeDX != 0 {
		t.Fatalf("short drag did not settle: dx %v settling %v", a.swipeDX, a.swipeSettling)
	}
}

// 3. 中间点击显示/隐藏浮动工具栏。
func TestCompactToolbarToggle(t *testing.T) {
	_, _, tt := newTestReader(t)
	if tt.HasText("目录") {
		t.Fatal("toolbar visible before a tap")
	}
	tt.ClickAt(195, 400)
	tt.Frame()
	if !tt.HasText("目录") || !tt.HasText("设置") {
		t.Fatalf("toolbar not shown; texts %q", tt.Texts())
	}
	tt.ClickAt(195, 400)
	tt.Frame()
	if tt.HasText("目录") {
		t.Fatalf("toolbar still shown; texts %q", tt.Texts())
	}
}

// 4. 目录全屏列表跳转，跳到目标章并关闭目录。
func TestTOCJump(t *testing.T) {
	a, _, tt := newTestReader(t)
	tt.ClickAt(195, 400)
	tt.Frame()
	if err := tt.Click("目录"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.screen != screenTOC {
		t.Fatalf("screen %v, want TOC", a.screen)
	}
	if err := tt.Click("第二章 承"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.spine != 1 {
		t.Fatalf("spine %d, want 1", a.spine)
	}
	if a.screen != screenReader {
		t.Fatalf("TOC stayed open: %v", a.screen)
	}
}

// 5. 设置里改字号会触发一次新的排版。
func TestFontSizeRelayout(t *testing.T) {
	a, fe, tt := newTestReader(t)
	tt.ClickAt(195, 400)
	tt.Frame()
	if err := tt.Click("设置"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	before := a.style.FontScale
	calls := fe.layoutCount()
	if err := tt.Click("字号 +"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.style.FontScale <= before {
		t.Fatalf("font scale %v, want > %v", a.style.FontScale, before)
	}
	if fe.layoutCount() <= calls {
		t.Fatalf("layout calls %d, want > %d", fe.layoutCount(), calls)
	}
}

// 6. regular 宽窗口双页对开。
func TestRegularSpread(t *testing.T) {
	fe := newFakeEngine()
	a := NewReader(fe)
	a.Open("test.epub", nil)
	tt := NewTester(a.View, 1200, 800)
	tt.Frame()
	if a.columns != 2 || a.layoutKey.columns != 2 {
		t.Fatalf("columns %d/%d, want 2", a.columns, a.layoutKey.columns)
	}
	if len(a.layout.Pages) < 2 {
		t.Fatalf("pages %d, want >= 2", len(a.layout.Pages))
	}
	if got := a.pageLabel(); got != "1–2 / 3" {
		t.Fatalf("page label %q", got)
	}
	if len(a.bitmaps) != 2 {
		t.Fatalf("bitmaps %d, want 2", len(a.bitmaps))
	}
}

// 7. regular 键盘翻页和 ⌘/Ctrl +/- 调字号。
func TestKeyboard(t *testing.T) {
	fe := newFakeEngine()
	a := NewReader(fe)
	a.Open("test.epub", nil)
	tt := NewTester(a.View, 1200, 800)
	tt.Frame()
	tt.Key(0, KeyRight)
	tt.Frame()
	if a.page != 2 {
		t.Fatalf("right: page %d, want 2", a.page)
	}
	tt.Key(0, KeyLeft)
	tt.Frame()
	if a.page != 0 {
		t.Fatalf("left: page %d, want 0", a.page)
	}
	tt.Key(0, KeySpace)
	tt.Frame()
	if a.page != 2 {
		t.Fatalf("space: page %d, want 2", a.page)
	}
	scale := a.style.FontScale
	tt.Key(uiCmd, KeyEqual)
	tt.Frame()
	if a.style.FontScale <= scale {
		t.Fatalf("Cmd+= did not grow the font: %v", a.style.FontScale)
	}
}

// 8. resize 先防抖，期间保留旧页；150ms 后按新宽度重排。
func TestResizeDebounceKeepsOldPage(t *testing.T) {
	fe := newFakeEngine()
	a := NewReader(fe)
	a.Open("test.epub", nil)
	tt := NewTester(a.View, 1200, 800)
	tt.Frame()
	oldKey := a.layoutKey
	calls := fe.layoutCount()
	if len(a.bitmaps) == 0 {
		t.Fatal("no bitmaps before resize")
	}

	tt.SetSize(640, 900)
	if a.layoutKey != oldKey {
		t.Fatalf("layout changed before debounce: %+v -> %+v", oldKey, a.layoutKey)
	}
	if len(a.bitmaps) == 0 {
		t.Fatal("old pages dropped during relayout")
	}
	if a.columns != 1 {
		t.Fatalf("columns %d, want 1 after narrow resize", a.columns)
	}

	tt.Advance(200 * time.Millisecond)
	tt.Frame()
	if a.layoutKey == oldKey {
		t.Fatal("layout not redone after debounce")
	}
	if a.layoutKey.columns != 1 {
		t.Fatalf("new layout columns %d, want 1", a.layoutKey.columns)
	}
	if fe.layoutCount() <= calls {
		t.Fatalf("layout calls %d, want > %d", fe.layoutCount(), calls)
	}
}

// 9. 主题改夜间后窗口配色和引擎纸色都变。
func TestThemeNight(t *testing.T) {
	a, _, tt := newTestReader(t)
	tt.ClickAt(195, 400)
	tt.Frame()
	if err := tt.Click("设置"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if err := tt.Click("夜间"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.theme != ThemeNight {
		t.Fatalf("theme %v, want night", a.theme)
	}
	if a.style.Paper != 0x14161aff {
		t.Fatalf("paper %#x", a.style.Paper)
	}
}

// 10. regular 鼠标滚轮翻页。
func TestWheelTurnsPage(t *testing.T) {
	fe := newFakeEngine()
	a := NewReader(fe)
	a.Open("test.epub", nil)
	tt := NewTester(a.View, 1200, 800)
	tt.Frame()
	tt.Wheel(600, 400, 0, 1)
	tt.Advance(200 * time.Millisecond)
	if a.page != 2 {
		t.Fatalf("wheel: page %d, want 2", a.page)
	}
}
