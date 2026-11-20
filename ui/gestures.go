package ui

import . "github.com/egoist/mygo/ui"

// swipeGesture 记录一次按下到抬起。位置都相对于页面区域。
type swipeGesture struct {
	down  bool
	x, y  float32
	moved bool
}

// 松手超过页面宽度的 1/4 就翻页，否则回弹；小于这个距离算点击。
const (
	swipeFlipFraction = 0.25
	swipeTapSlop      = 8
)

// compactInput 是手机上的手势层：中间 1/3 切换工具栏，左右 1/3 翻页，
// 水平滑动跟手平移页面。
func (a *App) compactInput(width float32) func(InputEvent) bool {
	return func(ev InputEvent) bool {
		switch ev.Kind {
		case InputPointerDown:
			if ev.Button != 0 {
				return false
			}
			a.gesture = swipeGesture{down: true, x: ev.X, y: ev.Y}
			a.swipeDX, a.swipeSettling = 0, false
			return true

		case InputPointerMove:
			if !a.gesture.down || ev.Button != 0 {
				return false
			}
			dx, dy := ev.X-a.gesture.x, ev.Y-a.gesture.y
			if absF(dx) > 4 || absF(dy) > 4 {
				a.gesture.moved = true
			}
			// 横向手势才平移页面，纵向留给以后的滚动/选择。
			if absF(dx) >= absF(dy) {
				a.swipeDX = dx
			}
			return true

		case InputPointerUp:
			if !a.gesture.down {
				return false
			}
			g := a.gesture
			a.gesture.down = false
			dx := a.swipeDX
			a.swipeDX = 0
			if ev.Canceled {
				if dx != 0 {
					a.swipeSettling = true
				}
				return true
			}
			threshold := width * swipeFlipFraction
			switch {
			case dx <= -threshold:
				a.nextPage()
			case dx >= threshold:
				a.prevPage()
			case absF(dx) < swipeTapSlop && absF(ev.Y-g.y) < swipeTapSlop && !g.moved:
				a.tapAt(ev.X, width)
			default:
				a.swipeSettling = dx != 0
			}
			return true
		}
		return false
	}
}

// tapAt 是手机上的点击热区：左 1/3 上一页，右 1/3 下一页，
// 中间切换工具栏。
func (a *App) tapAt(x, width float32) {
	switch {
	case x < width/3:
		a.prevPage()
	case x > width*2/3:
		a.nextPage()
	default:
		a.toolbar = !a.toolbar
	}
}

// wheelInput 让桌面的滚轮/触控板滚动翻页。
func (a *App) wheelInput() func(InputEvent) bool {
	return func(ev InputEvent) bool {
		if ev.Kind != InputScroll {
			return false
		}
		switch {
		case ev.DY > 0:
			a.nextPage()
		case ev.DY < 0:
			a.prevPage()
		case ev.DX > 0:
			a.nextPage()
		case ev.DX < 0:
			a.prevPage()
		default:
			return false
		}
		return true
	}
}

func absF(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}
