package main

import (
	"testing"

	"github.com/Laic7092/ebook/ui"
	myui "github.com/egoist/mygo/ui"
)

// 没有引擎时窗口也能开：显示未接入引擎的占位页。
func TestShellWithoutEngine(t *testing.T) {
	a := ui.NewReader(nil)
	tt := myui.NewTester(a.View, 390, 844)
	tt.Frame()
	if !tt.HasText("尚未接入排版引擎") {
		t.Errorf("texts %q", tt.Texts())
	}
}
