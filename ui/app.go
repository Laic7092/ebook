package ui

import (
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/egoist/mygo"
	. "github.com/egoist/mygo/ui"
)

// 阅读器分两个阶段：还在打开/排版，或者可以显示。错误单独一档，
// 这样视图永远有东西可画，不会因为引擎缺失而白屏。
type readerPhase int

const (
	phaseEmpty readerPhase = iota
	phaseLoading
	phaseReady
	phaseError
)

// 面板状态。compact 用整屏面板，regular 只把设置放成浮层，
// 目录则是左侧栏，由 sidebar 控制。
type readerScreen int

const (
	screenReader readerScreen = iota
	screenTOC
	screenSettings
)

// wantPage 表示“排版完成后把当前页放在哪里”。wantKeep 表示尽量保持
// 原来的行位置（改字号、窗口缩放都走这条路）。
const (
	wantKeep   = iota // 保持当前行
	wantFirst         // 本章第一页
	wantEnd           // 本章最后一页
	wantByLine        // 跳到 wantFragment 所在页
	wantAt            // 跳到 wantIndex 指定的页（恢复上次阅读位置）
)

// 窗口尺寸变化后等这么久再重排，期间继续显示旧页。
const layoutDebounce = 150 * time.Millisecond

// layoutKey 是触发一次排版的全部输入。Style 都是可比较的，
// 因此 key 可以直接用 == 判断是否需要重排。
type layoutKey struct {
	spine    int
	viewport Rect
	columns  int
	style    Style
}

// layoutResult 从后台排版 job 带回 UI 线程。
type layoutResult struct {
	layout Layout
	err    error
	line   int
	lineOK bool
}

// rasterKey 唯一标识一张已光栅化的页。把 layoutKey 放进来，
// 窗口缩放或换字号后不会误用旧章节/旧尺寸的位图。
type rasterKey struct {
	layout layoutKey
	page   int
}

// App 是阅读器的全部持久状态。视图只读它，用户操作改它，然后
// MyGo 重建帧。App 不保存任何 Element。
type App struct {
	eng Engine
	win *mygo.Window

	// 引擎方法不是并发安全的，排版与光栅化可能同时在后台跑。
	mu sync.Mutex
	// 引擎里当前装着哪一份 layout；Bitmap 之前要确认它没被别的 job 换掉。
	engLayoutValid bool
	engLayoutKey   layoutKey

	phase    readerPhase
	errText  string
	book     Book
	toc      []TOCEntry
	baseFont float32

	// 阅读位置
	spine   int
	page    int
	columns int

	// 阅读设置
	style Style
	theme readingTheme

	// 已安装的排版结果（正在显示的那一份）
	hasLayout bool
	layout    Layout
	layoutKey layoutKey

	// 待安装的排版（包括 150ms 防抖期）
	pending    bool
	pendingKey layoutKey
	pendingAt  time.Time
	pendingGen int

	// 排版完成后要把页放到哪里
	wantPage     int
	wantLine     int
	wantFragment string

	// 页面位图缓存：只缓存当前 layoutKey 的可见页
	bitmaps   map[rasterKey]*Bitmap
	rastering map[rasterKey]bool
	failed    map[rasterKey]bool
	// fallback 是上一份 layout 正在显示的页。重排完成、新页还没
	// 光栅化好时先继续显示，不让窗口闪成空白。
	fallback []*Bitmap

	// compact 的滑动手势
	gesture       swipeGesture
	swipeDX       float32
	swipeSettling bool

	// 宿主提供的「打开文件」入口；为 nil 时界面上不显示打开按钮。
	openFn func()

	// 阅读位置变化时通知宿主，用来存「上次读到哪」。
	onMove              func(spine, page int)
	lastSpine, lastPage int

	// 恢复上次阅读位置：open 之后、第一次排版完成时用掉。
	restoreSpine, restorePage int
	hasRestore                bool
	wantIndex                 int

	// 界面开关
	screen  readerScreen
	sidebar bool // regular 的左侧目录栏
	toolbar bool // compact 的浮动工具栏
	mode    layoutMode
}

// NewReader 创建一个阅读器。eng 可以为 nil：这时显示“尚未接入引擎”，
// 方便在引擎落地前先把 UI 跑起来。
func NewReader(eng Engine) *App {
	a := &App{
		eng:       eng,
		phase:     phaseEmpty,
		style:     Style{FontScale: 1, LineScale: 1},
		theme:     ThemePaper,
		columns:   1,
		screen:    screenReader,
		sidebar:   true,
		wantPage:  wantKeep,
		bitmaps:   map[rasterKey]*Bitmap{},
		rastering: map[rasterKey]bool{},
		failed:    map[rasterKey]bool{},
	}
	a.applyPalette()
	return a
}

// SetWindow 让 App 能把后台 job 的结果回投到窗口。测试里不调用它，
// 所有“慢活”就同步执行，测试不需要等 goroutine。
func (a *App) SetWindow(w *mygo.Window) { a.win = w }

// SetOpenHandler 让界面上的「打开文件」按钮去问宿主要一本书。
// 打开的动作（系统对话框、最近打开的路径）都由宿主决定。
func (a *App) SetOpenHandler(fn func()) { a.openFn = fn }

// SetMoveHandler 在阅读位置变化时通知宿主（翻页、跳章、跳目录）。
func (a *App) SetMoveHandler(fn func(spine, page int)) { a.onMove = fn }

// Position 返回当前阅读位置，宿主可以存下来，下次用 Restore 接着读。
func (a *App) Position() (spine, page int) { return a.spine, a.page }

// Restore 记住一个阅读位置。书还没打开时先存着，打开后第一次排版完成
// 就会跳过去；已经打开就直接跳。
func (a *App) Restore(spine, page int) {
	a.restoreSpine, a.restorePage, a.hasRestore = spine, page, true
	if a.phase == phaseReady {
		a.applyRestore()
	}
}

func (a *App) applyRestore() {
	if !a.hasRestore {
		return
	}
	a.hasRestore = false
	for _, ch := range a.book.Chapters {
		if ch.Spine == a.restoreSpine {
			a.spine = a.restoreSpine
			break
		}
	}
	a.wantPage, a.wantIndex = wantAt, a.restorePage
}

// Open 打开一本书。Open 可能在后台解析，所以没有返回值：错误进入
// phaseError，由视图显示。
func (a *App) Open(name string, data []byte) {
	if a.eng == nil {
		a.phase, a.errText = phaseError, "尚未接入排版引擎"
		return
	}
	a.phase, a.errText = phaseLoading, ""
	a.dispatch(func() {
		a.mu.Lock()
		book, err := a.eng.Open(name, data)
		var base float32
		var toc []TOCEntry
		if err == nil {
			base = a.eng.BaseFontSize()
			toc = a.eng.TOC()
		}
		a.mu.Unlock()
		a.post(func() { a.finishOpen(book, toc, base, err) })
	})
}

func (a *App) finishOpen(book Book, toc []TOCEntry, base float32, err error) {
	if err != nil {
		a.phase, a.errText = phaseError, err.Error()
		return
	}
	if len(toc) == 0 {
		toc = book.TOC
	}
	if base <= 0 {
		base = 16
	}
	a.book, a.toc, a.baseFont = book, toc, base
	a.phase, a.errText = phaseReady, ""
	a.spine = a.firstSpine()
	a.page = 0
	a.wantPage, a.wantLine, a.wantFragment = wantKeep, 0, ""
	a.hasLayout = false
	a.layout = Layout{}
	a.layoutKey = layoutKey{}
	a.pending = false
	a.bitmaps = map[rasterKey]*Bitmap{}
	a.rastering = map[rasterKey]bool{}
	a.failed = map[rasterKey]bool{}
	a.fallback = nil
	a.applyRestore()
}

// dispatch 把慢活放后台；测试里（没有窗口）直接执行。
func (a *App) dispatch(fn func()) {
	if a.win == nil {
		fn()
		return
	}
	go fn()
}

// post 回到 UI 线程；测试里直接执行。
func (a *App) post(fn func()) {
	if a.win == nil {
		fn()
		return
	}
	a.win.Update(fn)
}

// View 是窗口内容。每个 build pass 都会调用它。
func (a *App) View(c *Context) {
	a.applyTheme(c)
	if a.phase != phaseReady {
		a.buildStatus(c)
		return
	}

	w, h := c.Size()
	top, right, bottom, left := c.SafeArea()
	w -= left + right
	h -= top + bottom
	mode := breakpoint(w)
	a.mode = mode
	box := measurePages(w, h, mode, a.sidebar)
	a.columns = box.columns

	// 键盘快捷键先处理，这样同一帧就能用新的页号排版/渲染。
	if !c.FocusTakesText() {
		a.handleKeys(c)
	}

	key := a.desiredKey(box)
	a.requestLayout(c, key)
	a.ensureRasters(c)

	if mode == layoutCompact {
		a.buildCompact(c, box)
	} else {
		a.buildRegular(c, box)
	}

	// 设置面板里的控件可能在这一帧改了字号/主题；再请求一次，
	// 下一帧就会按新设置排版。
	a.requestLayout(c, a.desiredKey(box))

	if a.onMove != nil && (a.spine != a.lastSpine || a.page != a.lastPage) {
		a.lastSpine, a.lastPage = a.spine, a.page
		a.onMove(a.spine, a.page)
	}
}

func (a *App) desiredKey(box pageBox) layoutKey {
	return layoutKey{
		spine:    a.spine,
		viewport: box.viewport,
		columns:  box.columns,
		style:    a.style,
	}
}

// requestLayout 决定“现在排”还是“等 150ms 再排”。窗口 resize 时
// 保留旧 layout 和旧位图，重排完成后再整体换掉。
func (a *App) requestLayout(c *Context, key layoutKey) {
	// 同一章里的目录 fragment 不需要重新排版，但要让 job 调一次
	// FragmentLine 才能算出页；这时不能因为 key 没变就提前返回。
	force := a.wantPage == wantByLine && a.wantFragment != "" && !a.pending
	if a.hasLayout && a.layoutKey == key && !force {
		return
	}
	if a.pending && a.pendingKey == key {
		if a.pendingAt.IsZero() || c.Now().Before(a.pendingAt) {
			return // 已在后台排版，或还在防抖
		}
	} else {
		now := c.Now()
		var delay time.Duration
		// 只有“同样内容、同样设置、只有窗口尺寸变了”才防抖；
		// 目录跳转要立刻响应。
		if !force && a.hasLayout && a.layoutKey.spine == key.spine && a.layoutKey.style == key.style {
			delay = layoutDebounce
		}
		a.pending = true
		a.pendingKey = key
		a.pendingGen++
		a.pendingAt = now.Add(delay)
		if delay > 0 {
			c.After(delay)
			return // 等防抖计时器；期间旧 layout 继续显示
		}
	}
	a.startLayout(c, key, a.pendingGen)
}

// startLayout 启动一次排版。引擎调用在锁内完成，结果用 Window.Update
// 回到 UI 线程；没有窗口时同步执行，测试因此是确定性的。
func (a *App) startLayout(c *Context, key layoutKey, gen int) {
	a.pendingAt = time.Time{} // 已开始，后续 build 不再重复启动
	fragment := a.wantFragment
	before := a.layoutKey

	work := func() layoutResult {
		a.mu.Lock()
		defer a.mu.Unlock()
		lay, err := a.eng.Layout(key.spine, key.viewport, key.style)
		res := layoutResult{layout: lay, err: err}
		if err == nil && fragment != "" {
			res.line, res.lineOK = a.eng.FragmentLine(fragment)
		}
		if err == nil {
			a.engLayoutValid, a.engLayoutKey = true, key
		}
		return res
	}

	if a.win == nil {
		a.finishLayout(key, gen, work())
		if a.layoutKey != before {
			c.Invalidate() // 这次排版发生在本次 build 内，再画一帧
		}
		return
	}
	go func() {
		res := work()
		a.win.Update(func() { a.finishLayout(key, gen, res) })
	}()
}

func (a *App) finishLayout(key layoutKey, gen int, res layoutResult) {
	if gen != a.pendingGen || key != a.pendingKey {
		return // 过期的结果
	}
	a.pending = false
	if res.err != nil {
		a.phase, a.errText = phaseError, res.err.Error()
		return
	}

	oldKey, oldLayout, oldPage := a.layoutKey, a.layout, a.page
	lay := res.layout
	// 同一章内重排（resize/字号）：记住旧页，新位图回来前继续显示。
	a.fallback = nil
	if oldKey.spine == key.spine && oldKey != key {
		slots := max(1, key.columns)
		fb := make([]*Bitmap, slots)
		for i := 0; i < max(1, oldKey.columns) && i < slots; i++ {
			fb[i] = a.bitmaps[rasterKey{layout: oldKey, page: oldPage + i}]
		}
		a.fallback = fb
	}
	a.layout, a.layoutKey, a.hasLayout = lay, key, true
	a.bitmaps = map[rasterKey]*Bitmap{}
	a.rastering = map[rasterKey]bool{}
	a.failed = map[rasterKey]bool{}

	total := len(lay.Pages)
	switch {
	case total == 0:
		a.page = 0
	case a.wantPage == wantFirst:
		a.page = 0
	case a.wantPage == wantEnd:
		a.page = max(0, total-key.columns)
	case a.wantPage == wantAt:
		a.page = a.wantIndex
	case a.wantPage == wantByLine:
		line := a.wantLine
		if res.lineOK {
			line = res.line
		}
		a.page = pageForLine(lay, line)
	case oldKey.spine == key.spine && len(oldLayout.Pages) > 0:
		// 同一章内改字号或缩放：按原来的行找新页。
		line := oldLayout.Pages[clamp(oldPage, 0, len(oldLayout.Pages)-1)].LineFrom
		a.page = pageForLine(lay, line)
	default:
		a.page = 0
	}
	a.page = clamp(a.page, 0, max(0, total-1))
	a.wantPage, a.wantLine, a.wantFragment = wantKeep, 0, ""
}

// handleKeys 处理键盘翻页和字号。只在焦点不输入文字时生效。
func (a *App) handleKeys(c *Context) {
	switch {
	case c.Shortcut(0, KeyRight), c.Shortcut(0, KeyPageDown), c.Shortcut(0, KeySpace):
		a.nextPage()
	case c.Shortcut(0, KeyLeft), c.Shortcut(0, KeyPageUp):
		a.prevPage()
	case c.Shortcut(uiCmd, KeyEqual):
		a.adjustFont(0.1)
	case c.Shortcut(uiCmd, KeyMinus):
		a.adjustFont(-0.1)
	}
}

// uiCmd 是 Command（macOS）或 Ctrl（其它平台）的别名，单独写出来
// 是为了让快捷键一眼看得懂。
var uiCmd = Cmd

func (a *App) nextPage() {
	if a.hasLayout && len(a.layout.Pages) > 0 {
		step := max(1, a.layoutKey.columns)
		if a.page+step < len(a.layout.Pages) {
			a.page += step
			return
		}
	}
	a.nextChapter()
}

func (a *App) prevPage() {
	if a.hasLayout && a.page > 0 {
		step := max(1, a.layoutKey.columns)
		a.page = max(0, a.page-step)
		return
	}
	a.prevChapter()
}

func (a *App) nextChapter() {
	i := a.chapterIndex(a.spine)
	if i+1 < len(a.book.Chapters) {
		a.spine = a.book.Chapters[i+1].Spine
		a.wantPage, a.wantLine, a.wantFragment = wantFirst, 0, ""
	}
}

func (a *App) prevChapter() {
	i := a.chapterIndex(a.spine)
	if i > 0 {
		a.spine = a.book.Chapters[i-1].Spine
		a.wantPage, a.wantLine, a.wantFragment = wantEnd, 0, ""
	}
}

func (a *App) firstSpine() int {
	if len(a.book.Chapters) > 0 {
		return a.book.Chapters[0].Spine
	}
	return 0
}

func (a *App) chapterIndex(spine int) int {
	for i, ch := range a.book.Chapters {
		if ch.Spine == spine {
			return i
		}
	}
	return 0
}

func (a *App) chapterTitle(spine int) string {
	i := a.chapterIndex(spine)
	if i < len(a.book.Chapters) && a.book.Chapters[i].Title != "" {
		return a.book.Chapters[i].Title
	}
	return fmt.Sprintf("第 %d 章", i+1)
}

// goTOC 跳到目录项。Fragment 要等目标章排版完成后才能换算成行，
// 所以先记下，在排版 job 里调用 FragmentLine。
func (a *App) goTOC(e TOCEntry) {
	if e.Spine < 0 {
		return // 封面、注释页等不在阅读顺序里
	}
	if e.Spine == a.spine && e.Fragment == "" {
		// 同章的目录项没有 fragment：直接回到本章第一页。
		a.page = 0
		a.screen = screenReader
		a.toolbar = false
		return
	}
	a.spine = e.Spine
	a.wantPage = wantByLine
	a.wantFragment = e.Fragment
	a.wantLine = 0
	a.screen = screenReader
	a.toolbar = false
}

// openTOC 按当前布局档位打开目录：手机上整屏，桌面上是左侧栏。
func (a *App) openTOC() {
	if a.mode == layoutCompact {
		a.screen = screenTOC
		a.toolbar = false
		return
	}
	a.sidebar = !a.sidebar
}

// openSettings 打开设置面板；两种布局共用同一批控件。
func (a *App) openSettings() {
	if a.screen == screenSettings {
		a.screen = screenReader
		return
	}
	a.screen = screenSettings
}

// adjustFont 按 10% 步进改字号，范围 60%..240%。
func (a *App) adjustFont(delta float32) {
	s := a.style.FontScale + delta
	s = float32(math.Round(float64(s)*10)) / 10
	a.style.FontScale = clampF(s, 0.6, 2.4)
}

// adjustLine 调整行距倍数。
func (a *App) adjustLine(delta float32) {
	s := a.style.LineScale + delta
	s = float32(math.Round(float64(s)*10)) / 10
	a.style.LineScale = clampF(s, 0.8, 2.0)
}

func (a *App) setTheme(th readingTheme) {
	if th < 0 || th > ThemeNight {
		return
	}
	a.theme = th
	a.applyPalette()
}

func (a *App) fontPoints() float32 {
	base := a.baseFont
	if base <= 0 {
		base = 16
	}
	return base * a.style.FontScale
}

func (a *App) progress() float64 {
	n := len(a.book.Chapters)
	if n == 0 {
		return 0
	}
	p := 0.0
	if len(a.layout.Pages) > 0 {
		p = float64(a.page+a.layoutKey.columns) / float64(len(a.layout.Pages))
		p = math.Min(1, math.Max(0, p))
	}
	return math.Min(1, math.Max(0, (float64(a.chapterIndex(a.spine))+p)/float64(n)))
}

func (a *App) pageLabel() string {
	total := len(a.layout.Pages)
	if total == 0 {
		return "—"
	}
	first := a.page + 1
	last := a.page + max(1, a.layoutKey.columns)
	last = min(last, total)
	if last > first {
		return fmt.Sprintf("%d–%d / %d", first, last, total)
	}
	return fmt.Sprintf("%d / %d", first, total)
}

// buildStatus 画打开中、空引擎和错误状态。
func (a *App) buildStatus(c *Context) {
	t := c.Theme()
	Column(c).Fill().Center().Gap(12).Padding(32).Children(func() {
		Text(c, "eBook").FontSize(26).Bold()
		switch a.phase {
		case phaseLoading:
			Text(c, "正在打开…").TextColor(t.TextMuted)
		case phaseError:
			Text(c, a.errText).TextColor(t.Danger).TextAlign(Center).MaxWidth(420)
		default:
			if a.eng == nil {
				Text(c, "尚未接入排版引擎").TextColor(t.TextMuted)
				Text(c, "把 ui.NewReader(engine) 接到 internal/engine 即可。").TextColor(t.TextMuted).FontSize(13)
				return
			}
			Text(c, "还没有打开书").TextColor(t.TextMuted)
			if a.openFn != nil {
				Button(c, "打开 EPUB / TXT").OnClick(a.openFn)
			}
			Text(c, "也可以把书拖进窗口，或在文件管理里选「用 eBook 打开」。").TextColor(t.TextMuted).FontSize(13).TextAlign(Center).MaxWidth(420)
		}
	})
}

// pageForLine 在排版结果里找到包含某一行的页。
func pageForLine(lay Layout, line int) int {
	if len(lay.Pages) == 0 {
		return 0
	}
	if line < 0 {
		return 0
	}
	for i, p := range lay.Pages {
		end := p.LineFrom + p.LineLen
		if line >= p.LineFrom && (p.LineLen == 0 || line < end) {
			return i
		}
	}
	return len(lay.Pages) - 1
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func clampF(v, lo, hi float32) float32 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
