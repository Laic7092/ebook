package main

import (
	"flag"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/Laic7092/ebook/internal/engine"
	"github.com/Laic7092/ebook/internal/reader"
	"github.com/Laic7092/ebook/ui"
	"github.com/egoist/mygo"
	myui "github.com/egoist/mygo/ui"
)

// saved 是 mygo 帮这个 App 存下来的状态：上次读的那本书和读到哪。
type saved struct {
	Path  string
	Spine int
	Page  int
}

func main() {
	// -open 让命令行直接带一本书启动（桌面端调试、以及系统的「用 eBook 打开」）。
	open := flag.String("open", "", "启动时打开的书（EPUB 或 TXT）")
	flag.Parse()

	app := ui.NewReader(reader.New(engine.New()))
	state := &saved{}
	mygo.KeepState(state)

	// 位置一变就更新 state，KeepState 在退出/后台时把它存起来。
	app.SetMoveHandler(func(spine, page int) {
		state.Spine, state.Page = spine, page
	})

	app.SetOpenHandler(func() { chooseBook(app, state) })
	mygo.App.OnOpenFile(func(path string) { openBook(app, state, path) })

	mygo.App.WhenReady(func() {
		w := mygo.NewWindow(mygo.WindowOptions{
			Title:     "eBook",
			Width:     1200,
			Height:    800,
			MinWidth:  320,
			MinHeight: 480,
			StateKey:  "reader",
			Content:   myui.View(app.View),
		})
		app.SetWindow(w)

		switch {
		case *open != "":
			openBook(app, state, *open)
		case state.Path != "":
			openBook(app, state, state.Path)
		}
	})

	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

// chooseBook 问系统要一本书。手机上系统给的是副本，所以先落到自己的书库里。
func chooseBook(app *ui.App, state *saved) {
	paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
		Parent:  mygo.FocusedWindow(),
		Title:   "打开书",
		Message: "支持 EPUB 和 TXT",
		Filters: []mygo.FileFilter{{
			Name:       "电子书",
			Extensions: []string{"epub", "txt"},
			MIMETypes:  []string{"application/epub+zip", "text/plain"},
		}},
	})
	if err != nil || len(paths) == 0 {
		return
	}
	openBook(app, state, paths[0])
}

// openBook 打开一个文件。同一本书接着上次的位置，换书就从头开始。
func openBook(app *ui.App, state *saved, path string) {
	path = keepCopy(app, path)
	data, err := os.ReadFile(path)
	if err != nil {
		log.Printf("打开 %s 失败：%v", path, err)
		return
	}
	if state.Path != path {
		state.Path, state.Spine, state.Page = path, 0, 0
	}
	app.Open(filepath.Base(path), data)
	app.Restore(state.Spine, state.Page)
}

// keepCopy 把书存到 App 自己的目录里，因为手机给的是临时副本，随时会被清理。
// 已经在书库里就直接用，不再复制。
func keepCopy(app *ui.App, path string) string {
	dir, err := libraryDir()
	if err != nil {
		return path
	}
	if filepath.Dir(path) == dir {
		return path
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return path
	}
	dst := filepath.Join(dir, filepath.Base(path))
	in, err := os.Open(path)
	if err != nil {
		return path
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return path
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return path
	}
	if err := out.Close(); err != nil {
		return path
	}
	return dst
}

// libraryDir 是 App 自己放书的目录：macOS/Linux/iOS 都在用户配置目录下。
func libraryDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "eBook", "library"), nil
}
