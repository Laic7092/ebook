package engine

import "errors"

// 响应里嵌套结构的解码。字段顺序与 ebook-abi/src/protocol.rs 一致。

func zigzag(v uint64) int64 {
	return int64(v>>1) ^ -int64(v&1)
}

func (r *rbuf) i32() (int32, error) {
	value, err := r.u64()
	return int32(zigzag(value)), err
}

func decodeBook(r *rbuf) (*Book, error) {
	book := &Book{}
	var err error
	readOpt := func() string {
		if err != nil {
			return ""
		}
		var value string
		value, _, err = r.optStr()
		return value
	}
	book.Title = readOpt()
	book.Language = readOpt()
	book.Author = readOpt()
	readOpt() // publisher
	readOpt() // identifier
	readOpt() // date
	readOpt() // description
	book.Cover = readOpt()
	if err != nil {
		return nil, err
	}

	count, err := r.u32()
	if err != nil {
		return nil, err
	}
	book.Chapters = make([]Chapter, 0, count)
	for i := uint32(0); i < count; i++ {
		var chapter Chapter
		var index uint32
		if index, err = r.u32(); err != nil {
			return nil, err
		}
		chapter.Spine = int(index)
		if chapter.ID, err = r.str(); err != nil {
			return nil, err
		}
		if chapter.Href, err = r.str(); err != nil {
			return nil, err
		}
		if _, err = r.str(); err != nil { // media_type
			return nil, err
		}
		if chapter.Linear, err = r.boolean(); err != nil {
			return nil, err
		}
		book.Chapters = append(book.Chapters, chapter)
	}

	count, err = r.u32()
	if err != nil {
		return nil, err
	}
	book.Fonts = make([]FontFace, 0, count)
	for i := uint32(0); i < count; i++ {
		var face FontFace
		if face.Family, err = r.str(); err != nil {
			return nil, err
		}
		var weight uint32
		if weight, err = r.u32(); err != nil {
			return nil, err
		}
		face.Weight = int(weight)
		if face.Href, err = r.str(); err != nil {
			return nil, err
		}
		book.Fonts = append(book.Fonts, face)
	}
	return book, nil
}

func decodeTOC(r *rbuf) ([]TOCEntry, error) {
	count, err := r.u32()
	if err != nil {
		return nil, err
	}
	entries := make([]TOCEntry, 0, count)
	for i := uint32(0); i < count; i++ {
		var entry TOCEntry
		if entry.Label, err = r.str(); err != nil {
			return nil, err
		}
		spine, err2 := r.i32()
		if err2 != nil {
			return nil, err2
		}
		entry.Spine = int(spine)
		fragment, present, err3 := r.optStr()
		if err3 != nil {
			return nil, err3
		}
		if present {
			entry.Fragment = fragment
		}
		depth, err4 := r.u32()
		if err4 != nil {
			return nil, err4
		}
		entry.Depth = int(depth)
		entries = append(entries, entry)
	}
	return entries, nil
}

func decodePages(r *rbuf) ([]PageMeta, error) {
	count, err := r.u32()
	if err != nil {
		return nil, err
	}
	pages := make([]PageMeta, 0, count)
	for i := uint32(0); i < count; i++ {
		var page PageMeta
		var index, lineFrom, lineLen uint32
		if index, err = r.u32(); err != nil {
			return nil, err
		}
		_ = index // 顺序即下标，`PageMeta` 不带页号
		if page.Width, err = r.f32(); err != nil {
			return nil, err
		}
		if page.Height, err = r.f32(); err != nil {
			return nil, err
		}
		if page.Top, err = r.f32(); err != nil {
			return nil, err
		}
		if page.Right, err = r.f32(); err != nil {
			return nil, err
		}
		if page.Bottom, err = r.f32(); err != nil {
			return nil, err
		}
		if page.Left, err = r.f32(); err != nil {
			return nil, err
		}
		if lineFrom, err = r.u32(); err != nil {
			return nil, err
		}
		if lineLen, err = r.u32(); err != nil {
			return nil, err
		}
		page.LineFrom = int(lineFrom)
		page.LineLen = int(lineLen)
		if page.Progress, err = r.f32(); err != nil {
			return nil, err
		}
		pages = append(pages, page)
	}
	return pages, nil
}

func decodeLoc(r *rbuf) (Location, error) {
	var loc Location
	var err error
	if loc.Page, err = r.u32(); err != nil {
		return loc, err
	}
	if loc.Line, err = r.u32(); err != nil {
		return loc, err
	}
	if loc.Byte, err = r.u32(); err != nil {
		return loc, err
	}
	return loc, nil
}

func decodeSelection(r *rbuf) (*Selection, error) {
	present, err := r.boolean()
	if err != nil || !present {
		return nil, err
	}
	selection := &Selection{}
	if selection.Start, err = decodeLoc(r); err != nil {
		return nil, err
	}
	if selection.End, err = decodeLoc(r); err != nil {
		return nil, err
	}
	count, err := r.u32()
	if err != nil {
		return nil, err
	}
	for i := uint32(0); i < count; i++ {
		var rect SelectionRect
		if rect.Page, err = r.u32(); err != nil {
			return nil, err
		}
		if rect.X, err = r.f32(); err != nil {
			return nil, err
		}
		if rect.Y, err = r.f32(); err != nil {
			return nil, err
		}
		if rect.W, err = r.f32(); err != nil {
			return nil, err
		}
		if rect.H, err = r.f32(); err != nil {
			return nil, err
		}
		selection.Rects = append(selection.Rects, rect)
	}
	if selection.Text, err = r.str(); err != nil {
		return nil, err
	}
	return selection, nil
}

func decodeTarget(r *rbuf) (*Link, error) {
	present, err := r.boolean()
	if err != nil || !present {
		return nil, err
	}
	var link Link
	spine, err := r.i32()
	if err != nil {
		return nil, err
	}
	link.Spine = int(spine)
	fragment, hasFragment, err := r.optStr()
	if err != nil {
		return nil, err
	}
	if hasFragment {
		link.Fragment = fragment
	}
	if link.External, err = r.boolean(); err != nil {
		return nil, err
	}
	return &link, nil
}

var errNoValue = errors.New("engine: 响应类型不符")
