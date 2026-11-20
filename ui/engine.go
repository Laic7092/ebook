package ui

import (
	"image"
	"sync"

	"github.com/egoist/mygo"
	myui "github.com/egoist/mygo/ui"
)

// ThemeMode 是阅读主题。
type ThemeMode int

const (
	// ThemePaper 是纸色主题。
	ThemePaper ThemeMode = iota
	// ThemeNight 是夜间主题。
	ThemeNight
)

// State 是跨窗口重建、跨 mygo dev 重启都保留的阅读状态。
// main 把它交给 mygo.KeepState，字段本身是纯数据。
type State struct {
	Spine     int
	Page      int
	FontScale float64 // 0.8 .. 1.6
	LineScale float64 // 0.9 .. 1.8
	Theme     ThemeMode
	Bold      bool
}

// App 是阅读器外壳的状态：书、当前章页、设置、覆盖层和渲染结果。
// 视图在 UI 线程读它，慢活由 worker 排版/光栅化后经 Window.Update 回投。
type App struct {
	// State 由 main 交给 mygo.KeepState 保存。
	State State

	// DeviceScale 是请求位图时用的设备像素比。MyGo 暂时没有公开
	// 屏幕缩放，先固定为 2；接上平台时改这里即可。
	DeviceScale float32

	eng  Engine
	book Book
	win  *mygo.Window
	post func(func())

	mu      sync.Mutex // 只保护 reqCh 的启动，不保护界面状态
	reqCh   chan renderReq
	pending bool
	gen     uint64

	layout         Layout
	layoutValid    bool
	layoutSpine    int
	layoutViewport Rect
	layoutStyle    Style

	bitmap      *myui.Bitmap
	bitmapPage  int
	bitmapSpine int
	bitmapStyle Style

	// 打开目录项后要跳到的 fragment；排版完成后由 worker 换算成页。
	pendingFragment string
	// 往前翻到上一章时，排版完成后停在最后一页。
	pendingLast bool

	lastErr string

	// 覆盖层与工具栏。
	chrome       bool
	tocOpen      bool
	settingsOpen bool
	tocList      myui.ListState
	tocChoice    int
}

// New 创建阅读器外壳。eng 可以为 nil：窗口仍会打开，显示“未打开书籍”，
// 设置面板也可以调。真实引擎由 main 传进来。
func New(eng Engine) *App {
	return &App{
		State: State{
			Spine:     -1,
			Page:      0,
			FontScale: 1,
			LineScale: 1,
			Theme:     ThemePaper,
		},
		DeviceScale: 2,
		eng:         eng,
		chrome:      true,
		tocChoice:   -1,
	}
}

// SetWindow 把窗口接到外壳上：慢活的结果经 window.Update 回到 UI 线程。
// 窗口创建后调用一次。
func (a *App) SetWindow(w *mygo.Window) {
	if a == nil {
		return
	}
	a.win = w
	a.post = w.Update
	a.mu.Lock()
	if a.reqCh == nil {
		a.reqCh = make(chan renderReq, 1)
		go a.work(a.reqCh, a.post)
	}
	a.mu.Unlock()
}

// Open 用引擎打开一本书。它不是并发安全的，所以在创建窗口前、
// UI 线程之外准备好数据再调用；窗口已经打开时用 Window.Update 回投。
func (a *App) Open(name string, data []byte) error {
	if a.eng == nil {
		return errNoEngine{}
	}
	book, err := a.eng.Open(name, data)
	if err != nil {
		return err
	}
	a.book = book
	a.State.Page = 0
	a.State.Spine = 0
	for _, ch := range book.Chapters {
		a.State.Spine = ch.Spine
		break
	}
	a.layoutValid = false
	a.pendingFragment = ""
	a.bitmap = nil
	a.tocChoice = -1
	a.lastErr = ""
	a.requestFrame()
	return nil
}

// Title 返回书名，没打开书时是窗口的名字。
func (a *App) Title() string {
	if a.book.Title != "" {
		return a.book.Title
	}
	return "eBook"
}

// Close 释放引擎会话。窗口关闭后调用。
func (a *App) Close() error {
	if a.eng == nil {
		return nil
	}
	return a.eng.Close()
}

func (a *App) requestFrame() {
	if a.win != nil {
		a.win.Invalidate()
	}
}

// normalize 给 KeepState 读回来的旧数据补默认值。
func (a *App) normalize() {
	if a.State.FontScale <= 0 {
		a.State.FontScale = 1
	}
	if a.State.LineScale <= 0 {
		a.State.LineScale = 1
	}
	if a.State.Page < 0 {
		a.State.Page = 0
	}
	if a.State.Theme != ThemeNight {
		a.State.Theme = ThemePaper
	}
	if a.State.Spine < 0 {
		a.State.Spine = 0
		for _, ch := range a.book.Chapters {
			a.State.Spine = ch.Spine
			break
		}
	}
	if a.DeviceScale <= 0 {
		a.DeviceScale = 2
	}
}

// style 是交给引擎的阅读设置；主题的纸色与墨色来自界面主题。
func (a *App) style() Style {
	th := a.uiTheme()
	return Style{
		FontScale:     float32(a.State.FontScale),
		LineScale:     float32(a.State.LineScale),
		Ink:           rgbaHex(th.Text),
		Paper:         rgbaHex(th.Background),
		Bold:          a.State.Bold,
		LetterSpacing: 0,
		Align:         0,
	}
}

func rgbaHex(c myui.Color) uint32 {
	return uint32(c.R)<<24 | uint32(c.G)<<16 | uint32(c.B)<<8 | uint32(c.A)
}

// pageInsets 是页面四周留出的空白：上下给工具栏和进度，左右是安全边距。
func (a *App) pageInsets(w float32) (top, right, bottom, left float32) {
	margin := w * 0.06
	if margin < 12 {
		margin = 12
	}
	if margin > 40 {
		margin = 40
	}
	top = 12
	if a.chrome {
		top = 60
	}
	return top, margin, 44, margin
}

// pageViewport 是给引擎排版用的可用区域（点）。上下给工具栏和进度让出
// 位置，左右留出安全边距，内容不会贴边。
func (a *App) pageViewport(w, h float32) Rect {
	if w <= 0 || h <= 0 {
		return Rect{}
	}
	top, right, bottom, left := a.pageInsets(w)
	vw := w - left - right
	vh := h - top - bottom
	if vw < 1 {
		vw = 1
	}
	if vh < 1 {
		vh = 1
	}
	return Rect{X: 0, Y: 0, W: vw, H: vh}
}

// ensureRender 检查当前章、页面区域和设置，必要时把排版/光栅化交给
// worker。c 只在同步模式（测试）里用来请求下一帧。
func (a *App) ensureRender(c *myui.Context, viewport Rect) {
	if a.eng == nil || len(a.book.Chapters) == 0 {
		return
	}
	if a.pending {
		return
	}
	st := a.style()
	needLayout := !a.layoutValid || a.layoutSpine != a.State.Spine ||
		a.layoutViewport != viewport || a.layoutStyle != st
	if !needLayout && len(a.layout.Pages) == 0 {
		needLayout = true
	}
	needBitmap := a.bitmap == nil || a.bitmapPage != a.State.Page ||
		a.bitmapSpine != a.State.Spine || a.bitmapStyle != st
	if !needLayout && !needBitmap && a.pendingFragment == "" {
		return
	}

	a.gen++
	req := renderReq{
		gen:        a.gen,
		spine:      a.State.Spine,
		page:       a.State.Page,
		viewport:   viewport,
		style:      st,
		scale:      a.DeviceScale,
		needLayout: needLayout,
		layout:     a.layout,
		fragment:   a.pendingFragment,
		last:       a.pendingLast,
	}
	a.pending = true

	a.mu.Lock()
	ch := a.reqCh
	post := a.post
	a.mu.Unlock()
	if ch == nil || post == nil {
		a.apply(req.run(a.eng), c)
		return
	}
	select {
	case ch <- req:
	default:
		// 只保留最新的一帧；旧结果会因 gen 不匹配被丢掉。
		select {
		case <-ch:
		default:
		}
		ch <- req
	}
}

func (a *App) work(ch <-chan renderReq, post func(func())) {
	for req := range ch {
		res := req.run(a.eng)
		post(func() { a.apply(res, nil) })
	}
}

// renderReq 是一帧排版/光栅化请求。它不持有任何 Element。
type renderReq struct {
	gen        uint64
	spine      int
	page       int
	viewport   Rect
	style      Style
	scale      float32
	needLayout bool
	layout     Layout
	fragment   string
	last       bool
}

type renderRes struct {
	gen      uint64
	layout   Layout
	page     int
	img      *image.RGBA
	fragment string
	spine    int
	viewport Rect
	style    Style
	last     bool
	err      error
}

func (r renderReq) run(eng Engine) renderRes {
	res := renderRes{
		gen: r.gen, layout: r.layout, page: r.page, fragment: r.fragment,
		spine: r.spine, viewport: r.viewport, style: r.style, last: r.last,
	}
	if eng == nil {
		return res
	}
	if r.needLayout {
		// UI 侧的 `Rect` 只是页面区域；引擎的 `Viewport` 还带内边距，
		// 这里内边距已经折算进 Rect，所以四边都传 0。
		viewport := Viewport{Width: r.viewport.W, Height: r.viewport.H}
		pages, err := eng.Layout(r.spine, viewport, r.style)
		if err != nil {
			res.err = err
			return res
		}
		res.layout = Layout{Spine: r.spine, Pages: pages}
	}
	if len(res.layout.Pages) == 0 {
		res.page = 0
		return res
	}
	if r.fragment != "" {
		if line, ok := eng.FragmentLine(r.fragment); ok {
			res.page = pageForLine(res.layout, line)
		} else {
			res.page = 0
		}
	} else if r.last {
		res.page = len(res.layout.Pages) - 1
	}
	if res.page < 0 {
		res.page = 0
	}
	if res.page >= len(res.layout.Pages) {
		res.page = len(res.layout.Pages) - 1
	}
	bm, err := eng.Bitmap(res.page, r.scale, r.style)
	if err != nil {
		res.err = err
		return res
	}
	res.img = bitmapImage(bm)
	return res
}

func pageForLine(layout Layout, line int) int {
	for i, p := range layout.Pages {
		if line < p.LineFrom+p.LineLen {
			return i
		}
	}
	return len(layout.Pages) - 1
}

// bitmapImage 把引擎的 RGBA 字节转成 image.RGBA，供 ui.NewBitmap 使用。
func bitmapImage(bm Bitmap) *image.RGBA {
	if bm.W <= 0 || bm.H <= 0 {
		return image.NewRGBA(image.Rect(0, 0, 0, 0))
	}
	img := image.NewRGBA(image.Rect(0, 0, bm.W, bm.H))
	n := copy(img.Pix, bm.RGBA)
	if n < len(img.Pix) {
		// 引擎给的字节不够时余下部分留白，别把旧像素画出来。
		for i := n; i < len(img.Pix); i++ {
			img.Pix[i] = 0xff
		}
	}
	return img
}

// apply 在 UI 线程接收 worker 的结果。
func (a *App) apply(res renderRes, c *myui.Context) {
	a.pending = false
	if res.gen != a.gen {
		return
	}
	if res.err != nil {
		a.lastErr = res.err.Error()
		if c != nil {
			c.Invalidate()
		}
		return
	}
	a.lastErr = ""
	a.layout = res.layout
	a.layoutValid = true
	a.layoutSpine = res.spine
	a.layoutViewport = res.viewport
	a.layoutStyle = res.style

	if len(a.layout.Pages) == 0 {
		a.bitmap = nil
		if c != nil {
			c.Invalidate()
		}
		return
	}
	if res.fragment != "" && res.fragment == a.pendingFragment {
		a.State.Page = res.page
		a.pendingFragment = ""
	} else if res.last && a.pendingLast {
		a.State.Page = res.page
		a.pendingLast = false
	} else if a.State.Page >= len(a.layout.Pages) {
		a.State.Page = len(a.layout.Pages) - 1
	}
	if res.img != nil {
		a.bitmap = myui.NewBitmap(res.img)
		a.bitmapPage = res.page
		a.bitmapSpine = res.spine
		a.bitmapStyle = res.style
	}
	if c != nil {
		c.Invalidate()
	}
}

type errNoEngine struct{}

func (errNoEngine) Error() string { return "没有引擎实现：请把 internal/engine 传给 ui.New" }
