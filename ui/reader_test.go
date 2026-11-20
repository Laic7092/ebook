package ui

import (
	"strconv"
	"testing"

	myui "github.com/egoist/mygo/ui"
)

func newTestReader(t *testing.T) (*App, *fakeEngine, *myui.Tester) {
	t.Helper()
	fe := newFakeEngine()
	a := New(fe)
	if err := a.Open("test.epub", nil); err != nil {
		t.Fatalf("open: %v", err)
	}
	tt := myui.NewTester(a.View, 390, 844)
	tt.Frame()
	if a.pageCount() == 0 {
		t.Fatalf("第一帧没有排版出页面：%q", tt.Texts())
	}
	return a, fe, tt
}

// 第一帧：书名、页码、整页位图都在。
func TestFirstFrameShowsPage(t *testing.T) {
	a, _, tt := newTestReader(t)
	if !tt.HasText("测试之书") {
		t.Fatalf("书名没画出来：%q", tt.Texts())
	}
	if !tt.HasText("1 / " + strconv.Itoa(a.pageCount())) {
		t.Fatalf("页码没画出来：%q", tt.Texts())
	}
	if a.bitmap == nil {
		t.Fatalf("没有光栅化出位图")
	}
}

// 点击左/右三分之一翻页，中间切换工具栏。
func TestTapZones(t *testing.T) {
	a, _, tt := newTestReader(t)

	tt.ClickAt(360, 420) // 右 1/3
	tt.Frame()
	if a.State.Page != 1 {
		t.Fatalf("右点之后 page = %d, want 1", a.State.Page)
	}
	if !tt.HasText("2 / " + strconv.Itoa(a.pageCount())) {
		t.Errorf("页码没更新：%q", tt.Texts())
	}

	tt.ClickAt(20, 420) // 左 1/3
	tt.Frame()
	if a.State.Page != 0 {
		t.Fatalf("左点之后 page = %d, want 0", a.State.Page)
	}

	before := a.chrome
	tt.ClickAt(195, 420) // 中间
	tt.Frame()
	if a.chrome == before {
		t.Errorf("中间点击没有切换工具栏")
	}
}

// 水平滑动超过阈值翻页；竖直方向的拖动不算。
func TestSwipe(t *testing.T) {
	a, _, tt := newTestReader(t)

	tt.Press(340, 420)
	tt.Move(240, 421)
	tt.Move(60, 422)
	tt.Release(60, 422)
	tt.Frame()
	if a.State.Page != 1 {
		t.Fatalf("左滑 page = %d, want 1", a.State.Page)
	}

	tt.Press(60, 300)
	tt.Move(70, 420)
	tt.Release(70, 420)
	tt.Frame()
	if a.State.Page != 1 {
		t.Errorf("竖直拖动不该翻页，page = %d", a.State.Page)
	}
}

// 键盘翻页。
func TestKeyboard(t *testing.T) {
	a, _, tt := newTestReader(t)

	tt.Key(0, myui.KeyRight)
	tt.Frame()
	if a.State.Page != 1 {
		t.Fatalf("→ page = %d, want 1", a.State.Page)
	}
	tt.Key(0, myui.KeyLeft)
	tt.Frame()
	if a.State.Page != 0 {
		t.Fatalf("← page = %d, want 0", a.State.Page)
	}
	tt.Key(0, myui.KeySpace)
	tt.Frame()
	if a.State.Page != 1 {
		t.Fatalf("空格 page = %d, want 1", a.State.Page)
	}
}

// 目录：打开、点条目跳章，fragment 被换算成页。
func TestTOCJump(t *testing.T) {
	a, _, tt := newTestReader(t)

	a.openTOC()
	tt.Frame()
	if !tt.HasText("目录") {
		t.Fatalf("目录没打开：%q", tt.Texts())
	}
	if err := tt.Click("第二章 下雨"); err != nil {
		t.Fatalf("点目录项: %v", err)
	}
	tt.Frame()
	if a.State.Spine != 1 {
		t.Fatalf("跳转后 spine = %d, want 1", a.State.Spine)
	}
	if a.tocOpen {
		t.Errorf("跳转之后目录没关")
	}
	// 假引擎说 fragment "ch2" 在第 25 行，每页 20 行 → 第 2 页。
	if a.State.Page != 1 {
		t.Errorf("跳转后 page = %d, want 1", a.State.Page)
	}
}

// 改设置会触发重新排版，并且页码回到合理位置。
func TestSettingsTriggerRelayout(t *testing.T) {
	a, fe, tt := newTestReader(t)

	before := fe.layoutCalls
	a.settingsOpen = true
	tt.Frame()
	if err := tt.Click("增大字号"); err != nil {
		t.Fatalf("点增大字号: %v", err)
	}
	tt.Frame()
	if a.State.FontScale <= 1 {
		t.Fatalf("字号没变：%v", a.State.FontScale)
	}
	if fe.layoutCalls <= before {
		t.Errorf("改字号没有触发重排：%d -> %d", before, fe.layoutCalls)
	}
	if a.State.Page < 0 || a.State.Page >= a.pageCount() {
		t.Errorf("重排之后页号越界：%d / %d", a.State.Page, a.pageCount())
	}
}

// 章尾翻页进入下一章；章首回翻进入上一章并停在最后一页。
func TestChapterTurn(t *testing.T) {
	a, _, tt := newTestReader(t)

	a.State.Page = a.pageCount() - 1
	tt.Frame()
	a.nextPage()
	tt.Frame()
	if a.State.Spine != 1 || a.State.Page != 0 {
		t.Fatalf("章尾翻页后 spine/page = %d/%d, want 1/0", a.State.Spine, a.State.Page)
	}

	a.previousPage()
	tt.Frame()
	if a.State.Spine != 0 {
		t.Fatalf("章首回翻后 spine = %d, want 0", a.State.Spine)
	}
	if a.State.Page != a.pageCount()-1 {
		t.Fatalf("回翻应停在上一章最后一页：%d / %d", a.State.Page, a.pageCount())
	}
}

// 旧状态（比如改过默认值）会被 normalize 修正。
func TestNormalizeState(t *testing.T) {
	a, _, _ := newTestReader(t)
	a.State.FontScale = 0
	a.State.LineScale = -1
	a.State.Page = -5
	a.State.Theme = 9
	a.normalize()
	if a.State.FontScale != 1 || a.State.LineScale != 1 {
		t.Errorf("字号/行距没被修正：%v %v", a.State.FontScale, a.State.LineScale)
	}
	if a.State.Page != 0 || a.State.Theme != ThemePaper {
		t.Errorf("页号/主题没被修正：%v %v", a.State.Page, a.State.Theme)
	}
}
