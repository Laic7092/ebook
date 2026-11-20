package engine

// postcard 编解码，只覆盖 ebook-abi 用到的形状。
//
// 规则（与 Rust postcard crate 一致）：
//   - 无符号整数：LEB128 变长
//   - 有符号整数：zigzag 之后按 LEB128
//   - f32/f64：定长小端
//   - bool：一个字节 0/1
//   - Option：0 或 1 + 载荷
//   - Vec/String：长度（varint） + 元素
//   - 结构体：字段顺序拼接，没有标签
//   - 枚举：变体序号（varint） + 载荷
//
// 对应的 Rust 定义在 ebook-abi/src/protocol.rs，形状改了这里要一起改；
// 变异测试见 wire_test.go 里的固定字节。

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
)

// 请求变体序号，顺序与 protocol.rs 的 `Request` 一致。
const (
	reqLayout = iota
	reqBitmap
	reqHit
	reqLinkAt
	reqSelection
	reqProgress
	reqFragmentLine
	reqResolveLink
	reqRegisterFont
	reqRegisterFontFace
	reqBaseFontSize
	reqBook
	reqTOC
	reqPages
	reqLimits
	reqPing
)

// 响应变体序号，顺序与 protocol.rs 的 `Response` 一致。
const (
	respOk = iota
	respPong
	respBook
	respTOC
	respPages
	respBitmap
	respHit
	respLink
	respSelection
	respProgress
	respLine
	respTarget
	respFont
	respNumber
	respError
)

var errShort = errors.New("postcard: 数据被截断")

type wbuf struct{ b []byte }

func (w *wbuf) u64(v uint64) {
	for v >= 0x80 {
		w.b = append(w.b, byte(v)|0x80)
		v >>= 7
	}
	w.b = append(w.b, byte(v))
}

func (w *wbuf) u32(v uint32)  { w.u64(uint64(v)) }
func (w *wbuf) u8(v uint8)    { w.b = append(w.b, v) }
func (w *wbuf) boolean(v bool) {
	if v {
		w.b = append(w.b, 1)
	} else {
		w.b = append(w.b, 0)
	}
}

func (w *wbuf) f32(v float32) {
	var raw [4]byte
	binary.LittleEndian.PutUint32(raw[:], math.Float32bits(v))
	w.b = append(w.b, raw[:]...)
}

func (w *wbuf) bytes(v []byte) {
	w.u32(uint32(len(v)))
	w.b = append(w.b, v...)
}

func (w *wbuf) str(v string) {
	w.u32(uint32(len(v)))
	w.b = append(w.b, v...)
}

func (w *wbuf) some(ok bool) { w.u8(boolByte(ok)) }

func boolByte(v bool) uint8 {
	if v {
		return 1
	}
	return 0
}

type rbuf struct {
	b   []byte
	pos int
}

func (r *rbuf) u64() (uint64, error) {
	var value uint64
	var shift uint
	for {
		if r.pos >= len(r.b) {
			return 0, errShort
		}
		byteValue := r.b[r.pos]
		r.pos++
		value |= uint64(byteValue&0x7f) << shift
		if byteValue&0x80 == 0 {
			return value, nil
		}
		shift += 7
		if shift > 63 {
			return 0, errors.New("postcard: 变长整数过长")
		}
	}
}

func (r *rbuf) u32() (uint32, error) {
	value, err := r.u64()
	return uint32(value), err
}

func (r *rbuf) u8() (uint8, error) {
	if r.pos >= len(r.b) {
		return 0, errShort
	}
	value := r.b[r.pos]
	r.pos++
	return value, nil
}

func (r *rbuf) boolean() (bool, error) {
	value, err := r.u8()
	if err != nil {
		return false, err
	}
	switch value {
	case 0:
		return false, nil
	case 1:
		return true, nil
	default:
		return false, errors.New("postcard: 非法的 bool")
	}
}

func (r *rbuf) f32() (float32, error) {
	if r.pos+4 > len(r.b) {
		return 0, errShort
	}
	value := math.Float32frombits(binary.LittleEndian.Uint32(r.b[r.pos:]))
	r.pos += 4
	return value, nil
}

func (r *rbuf) str() (string, error) {
	size, err := r.u32()
	if err != nil {
		return "", err
	}
	if r.pos+int(size) > len(r.b) {
		return "", errShort
	}
	value := string(r.b[r.pos : r.pos+int(size)])
	r.pos += int(size)
	return value, nil
}

func (r *rbuf) bytes() ([]byte, error) {
	size, err := r.u32()
	if err != nil {
		return nil, err
	}
	if r.pos+int(size) > len(r.b) {
		return nil, errShort
	}
	value := make([]byte, size)
	copy(value, r.b[r.pos:r.pos+int(size)])
	r.pos += int(size)
	return value, nil
}

// optStr 解 `Option<String>`。
func (r *rbuf) optStr() (string, bool, error) {
	present, err := r.boolean()
	if err != nil || !present {
		return "", false, err
	}
	value, err := r.str()
	return value, true, err
}

func (r *rbuf) optF32() (float32, bool, error) {
	present, err := r.boolean()
	if err != nil || !present {
		return 0, false, err
	}
	value, err := r.f32()
	return value, true, err
}

func (r *rbuf) optU32() (uint32, bool, error) {
	present, err := r.boolean()
	if err != nil || !present {
		return 0, false, err
	}
	value, err := r.u32()
	return value, true, err
}

// ---------------------------------------------------------------- 请求

func encodeViewport(w *wbuf, v Viewport) {
	w.f32(v.Width)
	w.f32(v.Height)
	// Insets: top/right/bottom/left
	w.f32(v.Top)
	w.f32(v.Right)
	w.f32(v.Bottom)
	w.f32(v.Left)
}

func encodeStyle(w *wbuf, s Style) {
	w.f32(s.FontScale)
	w.f32(s.LineScale)
	w.f32(s.LetterSpacing)
	w.u32(uint32(s.Align))
}

func encodeLoc(w *wbuf, loc Location) {
	w.u32(loc.Page)
	w.u32(loc.Line)
	w.u32(loc.Byte)
}

func encodeLayout(spine int, viewport Viewport, style Style) []byte {
	w := &wbuf{}
	w.u32(reqLayout)
	w.u32(uint32(spine))
	encodeViewport(w, viewport)
	encodeStyle(w, style)
	return w.b
}

func encodeBitmap(page int, scale float32, style Style) []byte {
	w := &wbuf{}
	w.u32(reqBitmap)
	w.u32(uint32(page))
	w.f32(scale)
	w.u32(style.Ink)
	w.u32(style.Paper)
	w.boolean(style.Bold)
	return w.b
}

func encodePoint(kind int, page int, x, y float32) []byte {
	w := &wbuf{}
	w.u32(uint32(kind))
	w.u32(uint32(page))
	w.f32(x)
	w.f32(y)
	return w.b
}

func encodeSelection(a, b Location) []byte {
	w := &wbuf{}
	w.u32(reqSelection)
	encodeLoc(w, a)
	encodeLoc(w, b)
	return w.b
}

func encodeLocRequest(kind int, loc Location) []byte {
	w := &wbuf{}
	w.u32(uint32(kind))
	encodeLoc(w, loc)
	return w.b
}

func encodeStringRequest(kind int, value string) []byte {
	w := &wbuf{}
	w.u32(uint32(kind))
	w.str(value)
	return w.b
}

func encodeSimple(kind int) []byte {
	w := &wbuf{}
	w.u32(uint32(kind))
	return w.b
}

func encodeFont(data []byte) []byte {
	w := &wbuf{}
	w.u32(reqRegisterFont)
	w.bytes(data)
	return w.b
}

func encodeFontFace(index int) []byte {
	w := &wbuf{}
	w.u32(reqRegisterFontFace)
	w.u32(uint32(index))
	return w.b
}

// ---------------------------------------------------------------- 响应

type response struct {
	kind      uint32
	book      *Book
	toc       []TOCEntry
	pages     []PageMeta
	bitmap    *Bitmap
	hit       *Location
	link      *string
	selection *Selection
	progress  *float32
	line      *int
	target    *Link
	font      uint32
	number    float32
	message   string
}

func decodeResponse(data []byte) (response, error) {
	r := &rbuf{b: data}
	kind, err := r.u32()
	if err != nil {
		return response{}, err
	}
	out := response{kind: kind}
	switch kind {
	case respOk, respPong:
	case respError:
		out.message, err = r.str()
	case respBook:
		out.book, err = decodeBook(r)
	case respTOC:
		out.toc, err = decodeTOC(r)
	case respPages:
		out.pages, err = decodePages(r)
	case respBitmap:
		var width, height uint32
		if width, err = r.u32(); err == nil {
			if height, err = r.u32(); err == nil {
				var rgba []byte
				if rgba, err = r.bytes(); err == nil {
					out.bitmap = &Bitmap{W: int(width), H: int(height), RGBA: rgba}
				}
			}
		}
	case respHit:
		var page, line, byte_ uint32
		var present bool
		if present, err = r.boolean(); err == nil && present {
			if page, err = r.u32(); err == nil {
				if line, err = r.u32(); err == nil {
					if byte_, err = r.u32(); err == nil {
						out.hit = &Location{Page: page, Line: line, Byte: byte_}
					}
				}
			}
		}
	case respLink:
		value, present, err2 := r.optStr()
		if err2 == nil && present {
			out.link = &value
		}
		err = err2
	case respSelection:
		out.selection, err = decodeSelection(r)
	case respProgress:
		value, present, err2 := r.optF32()
		if err2 == nil && present {
			out.progress = &value
		}
		err = err2
	case respLine:
		value, present, err2 := r.optU32()
		if err2 == nil && present {
			line := int(value)
			out.line = &line
		}
		err = err2
	case respTarget:
		out.target, err = decodeTarget(r)
	case respFont:
		out.font, err = r.u32()
	case respNumber:
		out.number, err = r.f32()
	default:
		return response{}, fmt.Errorf("postcard: 未知的响应变体 %d", kind)
	}
	if err != nil {
		return response{}, err
	}
	if kind == respError {
		return out, nil
	}
	return out, nil
}
