package main

import (
	"testing"

	"github.com/Laic7092/ebook/ui"
	myui "github.com/egoist/mygo/ui"
)

// 没有打开书时窗口也能开：显示未打开书籍的占位页。
func TestShellWithoutEngine(t *testing.T) {
	a := ui.New(nil)
	tt := myui.NewTester(a.View, 390, 844)
	tt.Frame()
	if !tt.HasText("未打开书籍") {
		t.Errorf("texts %q", tt.Texts())
	}
}
