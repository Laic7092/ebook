//go:build !cgo

package engine

// 没有 cgo 的构建（比如纯 Go 的 CI、或还没交叉编译引擎的平台）退化成桩：
// UI 仍然能编译、能跑测试，只是打开书会明确报错，而不是给一个空页面。

type stub struct{}

// New 返回“引擎不可用”的桩。
func New() Engine { return stub{} }

func (stub) Open(string, []byte) (Book, error)               { return Book{}, ErrUnsupported }
func (stub) Layout(int, Viewport, Style) ([]PageMeta, error) { return nil, ErrUnsupported }
func (stub) Bitmap(int, float32, Style) (Bitmap, error)      { return Bitmap{}, ErrUnsupported }
func (stub) Hit(int, float32, float32) (Location, bool)      { return Location{}, false }
func (stub) LinkAt(int, float32, float32) (string, bool)     { return "", false }
func (stub) Selection(Location, Location) (Selection, bool)  { return Selection{}, false }
func (stub) Progress(Location) (float32, bool)               { return 0, false }
func (stub) FragmentLine(string) (int, bool)                 { return 0, false }
func (stub) ResolveLink(string) (Link, bool)                 { return Link{}, false }
func (stub) RegisterFont([]byte) (int, error)                { return 0, ErrUnsupported }
func (stub) RegisterFontFace(int) (int, error)               { return 0, ErrUnsupported }
func (stub) FontFaces() int                                  { return 0 }
func (stub) BaseFontSize() float32                           { return 16 }
func (stub) TOC() []TOCEntry                                 { return nil }
func (stub) Pages() []PageMeta                               { return nil }
func (stub) Close() error                                    { return nil }
