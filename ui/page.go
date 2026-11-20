package ui

import (
	"image"
	"time"

	. "github.com/egoist/mygo/ui"
)

// 页面位图按 2x 光栅化，再等比缩放到页面区域：在 1x 屏幕上略有过采样，
// 在 Retina/高 DPI 屏幕上不会糊。引擎的 Bitmap 只接受 1..3 的缩放比。
const rasterScale float32 = 2

// buildPages 画当前页（双页时两张并排）。所有页面都用 Fit(Contain)
// 等比缩放到自己的槽位，不拉伸，由 Row 居中。
func (a *App) buildPages(c *Context, box pageBox) {
	if !a.hasLayout {
		Column(c).Fill().Center().Children(func() {
			Text(c, "正在排版…").TextColor(c.Theme().TextMuted)
		})
		return
	}
	cols := max(1, a.layoutKey.columns)
	slotW := (box.area.W - pageGap*float32(cols-1)) / float32(cols)
	slotH := box.area.H
	slotW = maxF(1, slotW)
	slotH = maxF(1, slotH)

	Row(c).Fill().Gap(pageGap).Justify(Center).AlignItems(Center).Children(func() {
		for i := 0; i < cols; i++ {
			idx := a.page + i
			slot := Box(c.Key([2]int{a.layoutKey.spine, idx})).Width(slotW).Height(slotH)
			slot.Children(func() {
				if idx >= len(a.layout.Pages) {
					return
				}
				rk := rasterKey{layout: a.layoutKey, page: idx}
				switch {
				case a.bitmaps[rk] != nil:
					Image(c, a.bitmaps[rk]).Size(slotW, slotH).Fit(Contain).
						Label("第 " + itoa(idx+1) + " 页")
				case i < len(a.fallback) && a.fallback[i] != nil:
					// 新页还在光栅化：继续显示旧页，重排时不闪白。
					Image(c, a.fallback[i]).Size(slotW, slotH).Fit(Contain).
						Opacity(0.7).Label("正在重新排版")
				case a.failed[rk]:
					Text(c, "无法渲染此页").TextColor(c.Theme().Danger).FontSize(13)
				default:
					Text(c, "…").TextColor(c.Theme().TextMuted)
				}
			})
		}
	})
}

// ensureRasters 为当前可见页启动后台光栅化。只启动缺少的页，
// 结果回来之前视图继续显示旧位图或占位符。
func (a *App) ensureRasters(c *Context) {
	if !a.hasLayout || a.eng == nil || len(a.layout.Pages) == 0 {
		return
	}
	cols := max(1, a.layoutKey.columns)
	for i := 0; i < cols; i++ {
		idx := a.page + i
		if idx >= len(a.layout.Pages) {
			break
		}
		rk := rasterKey{layout: a.layoutKey, page: idx}
		if a.bitmaps[rk] != nil || a.rastering[rk] || a.failed[rk] {
			continue
		}
		a.rastering[rk] = true
		a.startRaster(rk)
	}
}

// startRaster 在后台调用引擎。引擎的 Bitmap 只对“当前 layout”有效，
// 所以进锁后先确认引擎里装的是这张 page 对应的 layout，不是就重排一次。
func (a *App) startRaster(rk rasterKey) {
	key, idx := rk.layout, rk.page
	work := func() (Raster, error) {
		a.mu.Lock()
		defer a.mu.Unlock()
		if !a.engLayoutValid || a.engLayoutKey != key {
			if _, err := a.eng.Layout(key.spine, key.viewport, key.style); err != nil {
				return Raster{}, err
			}
			a.engLayoutValid, a.engLayoutKey = true, key
		}
		return a.eng.Bitmap(idx, rasterScale, key.style)
	}

	if a.win == nil {
		r, err := work()
		a.finishRaster(rk, r, err)
		return
	}
	go func() {
		r, err := work()
		a.win.Update(func() { a.finishRaster(rk, r, err) })
	}()
}

// finishRaster 在 UI 线程把引擎像素变成 ui.Bitmap，缓存起来。
func (a *App) finishRaster(rk rasterKey, r Raster, err error) {
	delete(a.rastering, rk)
	if rk.layout != a.layoutKey {
		return // 排版已经换了，旧页丢掉
	}
	if err != nil || r.W <= 0 || r.H <= 0 || len(r.RGBA) < r.W*r.H*4 {
		a.failed[rk] = true
		return
	}
	img := &image.RGBA{
		Pix:    r.RGBA,
		Stride: r.W * 4,
		Rect:   image.Rect(0, 0, r.W, r.H),
	}
	a.bitmaps[rk] = NewBitmap(img)
	a.failed[rk] = false
	if a.visibleRastersReady() {
		a.fallback = nil
	}
}

// visibleRastersReady 报告当前屏上的页是否都已光栅化（失败也算，
// 失败会显示错误文字，不必再保留旧页）。
func (a *App) visibleRastersReady() bool {
	if !a.hasLayout {
		return true
	}
	cols := max(1, a.layoutKey.columns)
	for i := 0; i < cols; i++ {
		idx := a.page + i
		if idx >= len(a.layout.Pages) {
			break
		}
		rk := rasterKey{layout: a.layoutKey, page: idx}
		if a.bitmaps[rk] == nil && !a.failed[rk] {
			return false
		}
	}
	return true
}

// swipeOffset 返回页面容器的水平偏移。拖动时跟手；松手未过阈值时
// 用一个短缓动回弹到 0。
func (a *App) swipeOffset(e Element) float32 {
	if a.swipeSettling {
		off := e.Animate("page-swipe", 0, 120*time.Millisecond)
		if off > -0.5 && off < 0.5 {
			a.swipeSettling = false
			return 0
		}
		return off
	}
	return a.swipeDX
}

// itoa 只用于页码标签，避免为了一个整数把 fmt 拉进这个文件。
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
