package main

import (
	"log"

	"github.com/Laic7092/ebook/ui"
	"github.com/egoist/mygo"
	myui "github.com/egoist/mygo/ui"
)

func main() {
	// TODO: internal/engine 落地后在这里注入：ui.NewReader(engine.New())
	//           并把用户选中的 EPUB/TXT 通过 a.Open(name, data) 打开。
	a := ui.NewReader(nil)

	mygo.App.WhenReady(func() {
		w := mygo.NewWindow(mygo.WindowOptions{
			Title:     "eBook",
			Width:     1200,
			Height:    800,
			MinWidth:  320,
			MinHeight: 480,
			StateKey:  "reader",
			Content:   myui.View(a.View),
		})
		a.SetWindow(w)
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
