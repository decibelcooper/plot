package plt

import (
	"code.google.com/p/plotinum/vg"
	"code.google.com/p/plotinum/vg/veceps"
	"code.google.com/p/plotinum/vg/vecimg"
	"fmt"
	"image/color"
	"math"
	"strings"
)

// A drawArea is a vector graphics canvas along with
// an associated rect defining a section of the canvas
// to which drawing should take place.
type drawArea struct {
	vg.Canvas
	rect
}

// TextStyle describes what text will look like.
type TextStyle struct {
	// Color is the text color.
	Color color.Color

	// Font is the font description.
	Font vg.Font
}

// LineStyle describes what a line will look like.
type LineStyle struct {
	// Color is the color of the line.
	Color color.Color

	// Width is the width of the line.
	Width vg.Length

	Dashes   []vg.Length
	DashOffs vg.Length
}

// A GlyphShape is a lable representing a shape for drawing
// a glyph that represents a point.
//
// GlyphShape values that corresponding to uppercase ASCII
// letters (A'–'Z'), represent the shape of the corresponding
// character.  A handful of other GlyphShape values are
// defined as constants, all other GlyphShape values are
// invalid.
type GlyphShape uint8

const (
	// CircleGlyph is a filled circle
	CircleGlyph GlyphShape = iota

	// RingGlyph is an outlined circle
	RingGlyph
)

// A GlyphStyle specifies the look of a glyph used to draw
// a point on a plot.
type GlyphStyle struct {
	// Color is the color used to draw the glyph.
	color.Color

	// Shape is the shape of the glyph.
	Shape GlyphShape

	// Radius specifies the size of the glyph's radius.
	Radius vg.Length
}

// A glyphBox describes the location of a glyph
// and the offset/size of its bounding box.
type glyphBox struct {
	// The glyph location in normalized coordinates.
	x, y float64

	// rect is the offset of the glyph's minimum drawing
	// point relative to the glyph location and its size.
	rect
}

// NewDrawArea returns a new drawArea of a specified
// size using the given canvas.
func NewDrawArea(c vg.Canvas, w, h vg.Length) *drawArea {
	return &drawArea{Canvas: c, rect: rect{min: point{0, 0}, size: point{w, h}}}
}

// NewEPSDrawArea returns a new drawArea that saves to an
// encapsulated postscript file.
func NewEPSDrawArea(w, h vg.Length, title string) *drawArea {
	return NewDrawArea(veceps.New(w, h, title), w, h)
}

// NewPNGDrawArea makes a new drawArea that saves
// to a PNG file.
func NewPNGDrawArea(w, h vg.Length) (*drawArea, error) {
	img, err := vecimg.New(w, h)
	if err != nil {
		return nil, err
	}
	return NewDrawArea(img, w, h), nil
}

// center returns the center point of the area
func (da *drawArea) center() point {
	return point{
		x: (da.max().x-da.min.x)/2 + da.min.x,
		y: (da.max().y-da.min.y)/2 + da.min.y,
	}
}

// contains returns true if the drawArea contains the point.
func (da *drawArea) contains(p point) bool {
	return p.x <= da.max().x && p.x >= da.min.x &&
		p.y <= da.max().y && p.y >= da.min.y
}

// x returns the value of x, given in the unit range,
// in the drawing coordinates of this draw area.
// A value of 0, for example, will return the minimum
// x value of the draw area and a value of 1 will
// return the maximum.
func (da *drawArea) x(x float64) vg.Length {
	return vg.Length(x)*(da.max().x-da.min.x) + da.min.x
}

// y returns the value of x, given in the unit range,
// in the drawing coordinates of this draw area.
// A value of 0, for example, will return the minimum
// y value of the draw area and a value of 1 will
// return the maximum.
func (da *drawArea) y(y float64) vg.Length {
	return vg.Length(y)*(da.max().y-da.min.y) + da.min.y
}

// crop returns a new drawArea corresponding to the receiver
// area with the given number of inches added to the minimum
// and maximum x and y values of the drawArea's rect.
func (da *drawArea) crop(minx, miny, maxx, maxy vg.Length) *drawArea {
	minpt := point{
		x: da.rect.min.x + minx,
		y: da.rect.min.y + miny,
	}
	sz := point{
		x: da.max().x + maxx - minpt.x,
		y: da.max().y + maxy - minpt.y,
	}
	return &drawArea{
		vg.Canvas: vg.Canvas(da),
		rect:      rect{min: minpt, size: sz},
	}
}

// squishX returns a new drawArea with a squished width such
// that any of the given set of glyphs will draw within the original
// draw area when they are mapped to the coordinate system
// of the returned drawArea.
//
// The location of the glyphs that are given as a parameter are
// assumed to be on the unit interval, with 0 meaning the left-most
// side of the draw area and 1 meaning the right-most side.
func (da *drawArea) squishX(boxes []glyphBox) *drawArea {
	if len(boxes) == 0 {
		return da
	}

	boxes = append(boxes, glyphBox{}, glyphBox{x: 1})

	var left, right vg.Length
	minx, maxx := vg.Length(math.Inf(1)), vg.Length(math.Inf(-1))
	for _, b := range boxes {
		if x := da.x(b.x) + b.rect.min.x; x < minx {
			left = vg.Length(b.x)
			minx = x
		}
		if x := da.x(b.x) + b.min.x + b.size.x; x > maxx {
			right = vg.Length(b.x)
			maxx = x
		}
	}

	if minx >= da.min.x {
		minx = da.min.x
	}
	if maxx <= da.max().x {
		maxx = da.max().x
	}

	// where we want the left and right points to end up
	l := da.min.x + (da.min.x - minx)
	r := da.max().x - (maxx - da.max().x)
	n := (left*r - right*l) / (left - right)
	m := ((left-1)*r - right*l + l) / (left - right)
	return &drawArea{
		vg.Canvas: vg.Canvas(da),
		rect: rect{
			min:  point{x: n, y: da.min.y},
			size: point{x: m - n, y: da.size.y},
		},
	}
}

// squishY returns a new drawArea with a squished height such
// that any of the given set of glyphs will draw within the original
// draw area when they are mapped to the coordinate system
// of the returned drawArea.
//
// The location of the glyphs that are given as a parameter are
// assumed to be on the unit interval, with 0 meaning the
// bottom-most side of the draw area and 1 meaning the
// top-most side.
func (da *drawArea) squishY(boxes []glyphBox) *drawArea {
	if len(boxes) == 0 {
		return da
	}

	boxes = append(boxes, glyphBox{}, glyphBox{y: 1})

	var top, bot vg.Length
	miny, maxy := vg.Length(math.Inf(1)), vg.Length(math.Inf(-1))
	for _, b := range boxes {
		if y := da.y(b.y) + b.rect.min.y; y < miny {
			bot = vg.Length(b.y)
			miny = y
		}
		if y := da.y(b.y) + b.min.y + b.size.y; y > maxy {
			top = vg.Length(b.y)
			maxy = y
		}
	}

	if miny >= da.min.y {
		miny = da.min.y
	}
	if maxy <= da.max().y {
		maxy = da.max().y
	}

	// where we want the top and bottom points to end up
	b := da.min.y + (da.min.y - miny)
	t := da.max().y - (maxy - da.max().y)
	n := (bot*t - top*b) / (bot - top)
	m := ((bot-1)*t - top*b + b) / (bot - top)
	return &drawArea{
		vg.Canvas: vg.Canvas(da),
		rect: rect{
			min:  point{x: da.min.x, y: n},
			size: point{x: da.size.x, y: m - n},
		},
	}
}

// setLineStyle sets the current line style
func (da *drawArea) setLineStyle(sty LineStyle) {
	da.SetColor(sty.Color)
	da.SetLineWidth(sty.Width)
	var dashDots []vg.Length
	for _, dash := range sty.Dashes {
		dashDots = append(dashDots, dash)
	}
	da.SetLineDash(dashDots, sty.DashOffs)
}

// drawGlyph draws a glyph at a specified location.  If
// the location is outside of the drawArea then it is
// not drawn.
func drawGlyph(da *drawArea, sty GlyphStyle, pt point) {
	if !da.contains(pt) {
		return
	}

	da.setLineStyle(LineStyle{Width: vg.Points(0.5)})
	da.SetColor(sty.Color)

	switch {
	case sty.Shape == CircleGlyph:
		var p vg.Path
		p.Move(pt.x+sty.Radius, pt.y)
		p.Arc(pt.x, pt.y, sty.Radius, 0, 2*math.Pi)
		p.Close()
		da.Fill(p)

	case sty.Shape == RingGlyph:
		var p vg.Path
		p.Move(pt.x+sty.Radius, pt.y)
		p.Arc(pt.x, pt.y, sty.Radius, 0, 2*math.Pi)
		p.Close()
		da.Stroke(p)

	case sty.Shape >= 'A' && sty.Shape <= 'Z':
		font, err := vg.MakeFont(defaultFont, sty.Radius*2)
		if err != nil {
			panic(err)
		}
		str := string([]byte{byte(sty.Shape)})
		x := pt.x - font.Width(str)/2
		y := pt.y + font.Extents().Descent
		da.FillText(font, x, y, str)

	default:
		panic(fmt.Sprintf("Invalid GlyphShape: %d", sty.Shape))
	}
}

// drawLine draws a line connecting a set of points
// in the given drawArea.
func (da *drawArea) strokeLine(sty LineStyle, pts ...point) {
	if len(pts) == 0 {
		return
	}

	da.setLineStyle(sty)

	var p vg.Path
	p.Move(pts[0].x, pts[0].y)
	for _, pt := range pts {
		p.Line(pt.x, pt.y)
	}
	da.Stroke(p)
}

// strokeLine2 draws a line between two points in the given
// drawArea.
func (da *drawArea) strokeLine2(sty LineStyle, x0, y0, x1, y1 vg.Length) {
	da.strokeLine(sty, point{x0, y0}, point{x1, y1})
}

// strokeClippedLine draws a line that is clipped at the bounds
// the drawArea.
func (da *drawArea) strokeClippedLine(sty LineStyle, pts ...point) {
	// clip right
	lines0 := clip(isLeft, point{da.max().x, da.min.y}, point{-1, 0}, pts)

	// clip bottom
	var lines1 [][]point
	for _, line := range lines0 {
		ls := clip(isAbove, point{da.min.x, da.min.y}, point{0, -1}, line)
		lines1 = append(lines1, ls...)
	}

	// clip left
	lines0 = lines0[:0]
	for _, line := range lines1 {
		ls := clip(isRight, point{da.min.x, da.min.y}, point{1, 0}, line)
		lines0 = append(lines0, ls...)
	}

	// clip top
	lines1 = lines1[:0]
	for _, line := range lines0 {
		ls := clip(isBelow, point{da.min.x, da.max().y}, point{0, 1}, line)
		lines1 = append(lines1, ls...)
	}

	for _, l := range lines1 {
		da.strokeLine(sty, l...)
	}
	return
}

// clip performs clipping in a single clipping line specified
// by the norm, clip point, and in function.
func clip(in func(point, point) bool, clip, norm point, pts []point) (lines [][]point) {
	var l []point
	for i := 1; i < len(pts); i++ {
		cur, next := pts[i-1], pts[i]
		curIn, nextIn := in(cur, clip), in(next, clip)
		switch {
		case curIn && nextIn:
			l = append(l, cur)

		case curIn && !nextIn:
			l = append(l, cur, isect(cur, next, clip, norm))
			lines = append(lines, l)
			l = []point{}

		case !curIn && !nextIn:
			// do nothing

		default: // !curIn && nextIn
			l = append(l, isect(cur, next, clip, norm))
		}
		if nextIn && i == len(pts)-1 {
			l = append(l, next)
		}
	}
	if len(l) > 1 {
		lines = append(lines, l)
	}
	return
}

// slop is some slop for floating point equality
const slop = 3e-8 // ≈ √1⁻¹⁵

func isLeft(p, clip point) bool {
	return p.x <= clip.x+slop
}

func isRight(p, clip point) bool {
	return p.x >= clip.x-slop
}

func isBelow(p, clip point) bool {
	return p.y <= clip.y+slop
}

func isAbove(p, clip point) bool {
	return p.y >= clip.y-slop
}

// isect returns the intersection of a line p0→p1 with the
// clipping line specified by the clip point and normal.
func isect(p0, p1, clip, norm point) point {
	// t = (norm · (p0 - clip)) / (norm · (p0 - p1))
	t := p0.minus(clip).dot(norm) / p0.minus(p1).dot(norm)

	// p = p0 + t*(p1 - p0)
	return p1.minus(p0).scale(t).plus(p0)
}

// fillText fills lines of text in the draw area.
// The text is offset by its width times xalign and
// its height times yalign.  x and y give the bottom
// left corner of the text befor e it is offset.
func (da *drawArea) fillText(sty TextStyle, x, y vg.Length, xalign, yalign float64, txt string) {
	txt = strings.TrimRight(txt, "\n")
	if len(txt) == 0 {
		return
	}

	da.SetColor(sty.Color)

	ht := sty.height(txt)
	y += ht*vg.Length(yalign) - sty.Font.Extents().Ascent
	nl := textNLines(txt)
	for i, line := range strings.Split(txt, "\n") {
		xoffs := vg.Length(xalign) * sty.Font.Width(line)
		n := vg.Length(nl - i)
		da.FillText(sty.Font, x+xoffs, y+n*sty.Font.Size, line)
	}
}

// width returns the width of lines of text
// when using the given font.
func (sty TextStyle) width(txt string) (max vg.Length) {
	txt = strings.TrimRight(txt, "\n")
	for _, line := range strings.Split(txt, "\n") {
		if w := sty.Font.Width(line); w > max {
			max = w
		}
	}
	return
}

// height returns the height of the text when using
// the given font.
func (sty TextStyle) height(txt string) vg.Length {
	nl := textNLines(txt)
	if nl == 0 {
		return vg.Length(0)
	}
	e := sty.Font.Extents()
	return e.Height*vg.Length(nl-1) + e.Ascent
}

// textNLines returns the number of lines in the text.
func textNLines(txt string) int {
	txt = strings.TrimRight(txt, "\n")
	if len(txt) == 0 {
		return 0
	}
	n := 1
	for _, r := range txt {
		if r == '\n' {
			n++
		}
	}
	return n
}

// rectPath returns the path of a rectangle specified by its
// upper left corner, width and height.
func rectPath(r rect) (p vg.Path) {
	p.Move(r.min.x, r.min.y)
	p.Line(r.max().x, r.min.y)
	p.Line(r.max().x, r.max().y)
	p.Line(r.min.x, r.max().y)
	p.Close()
	return
}

// A rect represents a rectangular region of 2d space.
type rect struct {
	min, size point
}

// max returns the maxmium x and y values of a rect.
func (r rect) max() point {
	return point{
		x: r.min.x + r.size.x,
		y: r.min.y + r.size.y,
	}
}

// A point is a location in 2d space.
type point struct {
	x, y vg.Length
}

// dot returns the dot product of two points.
func (p point) dot(q point) vg.Length {
	return p.x*q.x + p.y*q.y
}

// plus returns the component-wise sum of two points.
func (p point) plus(q point) point {
	return point{p.x + q.x, p.y + q.y}
}

// minus returns the component-wise difference of two points.
func (p point) minus(q point) point {
	return point{p.x - q.x, p.y - q.y}
}

// scale returns the component-wise product of a point and a scalar.
func (p point) scale(s vg.Length) point {
	return point{p.x * s, p.y * s}
}