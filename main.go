package main

import (
	"flag"
	"log"
	"os"

	"github.com/Laic7092/ebook/internal/engine"
	"github.com/Laic7092/ebook/ui"
	"github.com/egoist/mygo"
	myui "github.com/egoist/mygo/ui"
)

func main() {
	var file string
	flag.StringVar(&file, "open", "", "启动时打开的 EPUB/TXT 文件")
	flag.Parse()
	if file == "" && flag.NArg() > 0 {
		file = flag.Arg(0)
	}

	app := ui.New(engine.New())

	// 页码、字号、行距、主题和加粗都在 State 里：切换窗口重建、
	// mygo dev 重启都读同一份。
	mygo.KeepState(&app.State)

	if file != "" {
		data, err := os.ReadFile(file)
		if err != nil {
			log.Printf("读不了 %s: %v", file, err)
		} else if err := app.Open(file, data); err != nil {
			log.Printf("打开 %s 失败: %v", file, err)
		}
	}

	mygo.App.WhenReady(func() {
		w := mygo.NewWindow(mygo.WindowOptions{
			Title:     "eBook",
			Width:     390,
			Height:    844,
			MinWidth:  320,
			MinHeight: 480,
			StateKey:  "reader",
			Content:   myui.View(app.View),
		})
		app.SetWindow(w)
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
