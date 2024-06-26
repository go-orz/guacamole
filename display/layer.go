package guacenc

import (
	"github.com/fogleman/gg"
	"image"
	"image/color"
	"image/draw"
)

func t() {
	//c := canvas.New(100, 200)
}

type layer struct {
	width        int
	height       int
	image        *image.RGBA
	gc           *gg.Context
	visible      bool
	modified     bool
	modifiedRect image.Rectangle
	pathOpen     bool
	pathRect     image.Rectangle
	autosize     bool
}

func (l *layer) updateModifiedRect(modArea image.Rectangle) {
	before := l.modifiedRect
	l.modifiedRect = l.modifiedRect.Union(modArea)
	l.modified = l.modified || !before.Eq(l.modifiedRect)
}

func (l *layer) resetModified() {
	l.modifiedRect = image.Rectangle{}
	l.modified = false
}

func (l *layer) setupCanvas() {
	l.gc = gg.NewContext(l.width, l.height)
}

func (l *layer) fitRect(x int, y int, w int, h int) {
	rect := image.Rect(x, y, x+w, y+h)
	final := l.image.Bounds().Union(rect)
	l.Resize(final.Max.X, final.Max.Y)
}

func copyImage(dest draw.Image, x, y int, src image.Image, sr image.Rectangle, op draw.Op) {
	dp := image.Pt(x, y)
	dr := image.Rectangle{Min: dp, Max: dp.Add(sr.Size())}
	draw.Draw(dest, dr, src, sr.Min, op)
}

func (l *layer) Copy(srcLayer *layer, srcx, srcy, srcw, srch, x, y int, op draw.Op) {
	srcImg := srcLayer.image
	srcDim := srcImg.Bounds()

	// If entire rectangle outside source canvas, stop
	if srcx >= srcDim.Max.X || srcy >= srcDim.Max.Y {
		return
	}

	// Otherwise, clip rectangle to area
	if srcx+srcw > srcDim.Max.X {
		srcw = srcDim.Max.X - srcx
	}

	if srcy+srch > srcDim.Max.Y {
		srch = srcDim.Max.Y - srcy
	}

	// Stop if nothing to draw.
	if srcw == 0 || srch == 0 {
		return
	}

	if l.autosize {
		l.fitRect(x, y, srcw, srch)
	}

	srcCopyDim := image.Rect(srcx, srcy, srcx+srcw, srcy+srch)
	copyImage(l.image, x, y, srcImg, srcCopyDim, op)
	l.updateModifiedRect(image.Rect(x, y, x+srcw, y+srch))
}

func (l *layer) Draw(x, y int, src image.Image, op draw.Op) {
	srcDim := src.Bounds()
	if l.autosize {
		l.fitRect(x, y, srcDim.Max.X, srcDim.Max.Y)
	}
	copyImage(l.image, x, y, src, srcDim, op)
	l.updateModifiedRect(image.Rect(x, y, x+srcDim.Max.X, y+srcDim.Max.Y))
}

func (l *layer) Resize(w int, h int) {
	original := l.image.Bounds()
	if w == l.width && h == l.height {
		return
	}
	newImage := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(newImage, l.image.Bounds(), l.image, image.Pt(0, 0), draw.Src)
	l.image = newImage
	l.width = w
	l.height = h
	l.setupCanvas()
	l.updateModifiedRect(original.Union(l.image.Bounds()))
}

func (l *layer) appendToPath(rect image.Rectangle) {
	if !l.pathOpen {
		//l.gc.BeginPath()
		l.pathOpen = true
		l.pathRect = image.Rectangle{}
	}
	l.pathRect = l.pathRect.Union(rect)
}

func (l *layer) endPath() {
	l.updateModifiedRect(l.pathRect)
	l.pathOpen = false
	l.pathRect = image.Rectangle{}
}

func (l *layer) Rect(x int, y int, width int, height int) {
	l.appendToPath(image.Rect(x, y, x+width, y+height))
	l.gc.DrawRectangle(float64(x), float64(y), float64(width), float64(height))
}

func (l *layer) Fill(r byte, g byte, b byte, a byte, op draw.Op) {
	// Ignores op, as the canvas library does not support it :/
	l.gc.SetFillStyle(gg.NewSolidPattern(color.RGBA{
		R: r,
		G: g,
		B: b,
		A: a,
	}))
	l.gc.Fill()
	l.endPath()
}

func (l *layer) Move(parent *layer, x, y, z int) {
	l.gc.Translate(float64(x), float64(y))
	// TODO 设置zindex
}

type layers map[int]*layer

func newLayers() layers {
	ls := make(layers)
	ls[0] = newBuffer()
	ls[0].visible = true
	return ls
}

func newBuffer() *layer {
	l := &layer{
		image:    image.NewRGBA(image.Rect(0, 0, 0, 0)),
		autosize: true,
	}
	l.setupCanvas()
	return l
}

func newVisibleLayer(l0 *layer) *layer {
	l := &layer{
		width:   l0.width,
		height:  l0.height,
		image:   image.NewRGBA(image.Rect(0, 0, l0.width, l0.height)),
		visible: true,
	}
	l.setupCanvas()
	return l
}

func (ls layers) getDefault() *layer {
	return ls[0]
}

func (ls layers) get(id int) *layer {
	if l, ok := ls[id]; ok {
		return l
	}
	if id > 0 {
		ls[id] = newVisibleLayer(ls[0])
	} else {
		ls[id] = newBuffer()
	}
	return ls[id]
}

func (ls layers) delete(id int) {
	if id == 0 {
		return
	}
	ls[0].updateModifiedRect(ls[id].image.Bounds())
	ls[id].image = nil
	ls[id] = nil
	delete(ls, id)
}
