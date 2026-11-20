package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/Laic7092/ebook/internal/engine"
	"github.com/Laic7092/ebook/internal/reader"
	"github.com/Laic7092/ebook/ui"
	"github.com/egoist/mygo"
	myui "github.com/egoist/mygo/ui"
)

func main() {
	// -open 让桌面端（以及本地调试）直接带一本书启动：
	//   ebook -open ~/Documents/demo.epub
	open := flag.String("open", "", "启动时打开的书（EPUB 或 TXT）")
	flag.Parse()

	app := ui.NewReader(reader.New(engine.New()))

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

		if *open != "" {
			data, err := os.ReadFile(*open)
			if err != nil {
				log.Printf("打开 %s 失败：%v", *open, err)
				return
			}
			app.Open(filepath.Base(*open), data)
		}
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
