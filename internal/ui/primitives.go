package ui

import (
	"fmt"
	"image"
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"

	textv2 "github.com/hajimehoshi/ebiten/v2/text/v2"
)

type Theme struct {
	BackdropTop    color.NRGBA
	BackdropBottom color.NRGBA
	PanelFill      color.NRGBA
	PanelFillAlt   color.NRGBA
	PanelShadow    color.NRGBA
	BorderDark     color.NRGBA
	BorderMid      color.NRGBA
	BorderLight    color.NRGBA
	Accent         color.NRGBA
	AccentMuted    color.NRGBA
	Text           color.NRGBA
	TextDim        color.NRGBA
	Danger         color.NRGBA
	Success        color.NRGBA
	InputFill      color.NRGBA
	MeterTrack     color.NRGBA
}

var RetroTheme = Theme{
	BackdropTop:    color.NRGBA{R: 18, G: 28, B: 52, A: 255},
	BackdropBottom: color.NRGBA{R: 8, G: 12, B: 24, A: 255},
	PanelFill:      color.NRGBA{R: 52, G: 42, B: 30, A: 235},
	PanelFillAlt:   color.NRGBA{R: 72, G: 58, B: 40, A: 245},
	PanelShadow:    color.NRGBA{R: 0, G: 0, B: 0, A: 120},
	BorderDark:     color.NRGBA{R: 28, G: 20, B: 12, A: 255},
	BorderMid:      color.NRGBA{R: 94, G: 76, B: 49, A: 255},
	BorderLight:    color.NRGBA{R: 210, G: 190, B: 135, A: 255},
	Accent:         color.NRGBA{R: 184, G: 142, B: 48, A: 255},
	AccentMuted:    color.NRGBA{R: 110, G: 86, B: 32, A: 255},
	Text:           color.NRGBA{R: 246, G: 232, B: 201, A: 255},
	TextDim:        color.NRGBA{R: 184, G: 170, B: 143, A: 255},
	Danger:         color.NRGBA{R: 170, G: 66, B: 58, A: 255},
	Success:        color.NRGBA{R: 78, G: 154, B: 96, A: 255},
	InputFill:      color.NRGBA{R: 28, G: 22, B: 17, A: 240},
	MeterTrack:     color.NRGBA{R: 24, G: 18, B: 14, A: 255},
}

type PanelOptions struct {
	Title  string
	Accent color.NRGBA
	Fill   color.NRGBA
}

// Font returns the legacy font.Face used for metrics calculations.
func Font() font.Face {
	return basicfont.Face7x13
}

var textFace = textv2.NewGoXFace(basicfont.Face7x13)

// fillRect wraps vector.FillRect with float64→float32 conversion.
func fillRect(dst *ebiten.Image, x, y, w, h float64, clr color.Color) {
	vector.FillRect(dst, float32(x), float32(y), float32(w), float32(h), clr, false)
}

// FillRect is the exported variant for use by other packages.
func FillRect(dst *ebiten.Image, x, y, w, h float64, clr color.Color) {
	fillRect(dst, x, y, w, h, clr)
}

func DrawBackdrop(screen *ebiten.Image, theme Theme, ticks int) {
	b := screen.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		t := float64(y-b.Min.Y) / float64(max(1, b.Dy()-1))
		c := blend(theme.BackdropTop, theme.BackdropBottom, t)
		fillRect(screen, float64(b.Min.X), float64(y), float64(b.Dx()), 1, c)
	}

	bandX := float64((ticks*2)%(b.Dx()+160) - 160)
	for i := 0; i < 4; i++ {
		x := bandX - float64(i*54)
		fillRect(screen, x, 0, 18, float64(b.Dy()), color.NRGBA{R: theme.Accent.R, G: theme.Accent.G, B: theme.Accent.B, A: 16})
	}

	for y := 0; y < b.Dy(); y += 4 {
		fillRect(screen, 0, float64(y), float64(b.Dx()), 1, color.NRGBA{R: 255, G: 255, B: 255, A: 10})
	}

	frame := image.Rect(12, 12, b.Max.X-12, b.Max.Y-12)
	DrawBorder(screen, frame, theme.BorderMid, theme.BorderDark, theme.BorderLight)
	DrawBorder(screen, frame.Inset(2), theme.BorderDark, theme.BorderDark, theme.AccentMuted)
}

func DrawPanel(screen *ebiten.Image, rect image.Rectangle, theme Theme, opts PanelOptions) {
	accent := opts.Accent
	if accent.A == 0 {
		accent = theme.Accent
	}
	fill := opts.Fill
	if fill.A == 0 {
		fill = theme.PanelFill
	}

	fillRect(screen, float64(rect.Min.X+4), float64(rect.Min.Y+4), float64(rect.Dx()), float64(rect.Dy()), theme.PanelShadow)
	fillRect(screen, float64(rect.Min.X), float64(rect.Min.Y), float64(rect.Dx()), float64(rect.Dy()), fill)
	fillRect(screen, float64(rect.Min.X+3), float64(rect.Min.Y+3), float64(rect.Dx()-6), float64(rect.Dy()-6), theme.PanelFillAlt)
	DrawBorder(screen, rect, theme.BorderDark, theme.BorderMid, theme.BorderLight)
	DrawBorder(screen, rect.Inset(2), theme.BorderDark, theme.BorderDark, accent)

	if opts.Title != "" {
		titleRect := image.Rect(rect.Min.X+18, rect.Min.Y-10, min(rect.Max.X-18, rect.Min.X+18+MeasureText(opts.Title)+24), rect.Min.Y+12)
		fillRect(screen, float64(titleRect.Min.X), float64(titleRect.Min.Y), float64(titleRect.Dx()), float64(titleRect.Dy()), theme.PanelFillAlt)
		DrawBorder(screen, titleRect, theme.BorderDark, theme.BorderMid, accent)
		DrawText(screen, opts.Title, titleRect.Min.X+12, titleRect.Min.Y+12, theme.Text)
	}
	DrawInnerShade(screen, rect.Inset(5), theme)
	drawCornerAccents(screen, rect.Inset(5), accent)
}

func DrawInset(screen *ebiten.Image, rect image.Rectangle, theme Theme, focused bool) {
	fillRect(screen, float64(rect.Min.X), float64(rect.Min.Y), float64(rect.Dx()), float64(rect.Dy()), theme.InputFill)
	DrawBorder(screen, rect, theme.BorderDark, theme.BorderMid, ternary(focused, theme.Accent, theme.BorderMid))
	if focused {
		fillRect(screen, float64(rect.Min.X+2), float64(rect.Min.Y+2), float64(rect.Dx()-4), 1, color.NRGBA{R: 255, G: 255, B: 255, A: 24})
	}
}

func DrawButton(screen *ebiten.Image, rect image.Rectangle, theme Theme, label string, active bool, disabled bool) {
	fill := theme.PanelFillAlt
	textColor := theme.Text
	accent := theme.BorderMid
	if active {
		fill = color.NRGBA{R: 86, G: 62, B: 34, A: 255}
		accent = theme.Accent
	}
	if disabled {
		fill = theme.PanelFill
		textColor = theme.TextDim
		accent = theme.BorderDark
	}
	fillRect(screen, float64(rect.Min.X), float64(rect.Min.Y), float64(rect.Dx()), float64(rect.Dy()), fill)
	if active && !disabled {
		hi := color.NRGBA{R: min(255, fill.R+20), G: min(255, fill.G+16), B: min(255, fill.B+10), A: 255}
		fillRect(screen, float64(rect.Min.X+2), float64(rect.Min.Y+2), float64(rect.Dx()-4), float64(rect.Dy()/2-1), hi)
		fillRect(screen, float64(rect.Min.X+2), float64(rect.Min.Y+2), float64(rect.Dx()-4), 1, color.NRGBA{R: 255, G: 255, B: 255, A: 20})
	}
	DrawBorder(screen, rect, theme.BorderDark, theme.BorderMid, accent)
	DrawTextCentered(screen, label, rect, textColor)
}

func DrawMeter(screen *ebiten.Image, rect image.Rectangle, theme Theme, label string, value float64, fill color.NRGBA) {
	if value < 0 {
		value = 0
	}
	if value > 1 {
		value = 1
	}
	fillRect(screen, float64(rect.Min.X), float64(rect.Min.Y), float64(rect.Dx()), float64(rect.Dy()), theme.MeterTrack)

	barW := float64(rect.Dx()-4) * value
	barH := float64(rect.Dy() - 4)
	barX := float64(rect.Min.X + 2)
	barY := float64(rect.Min.Y + 2)
	if barW > 1 {
		fillRect(screen, barX, barY, barW, barH, fill)
		highlight := color.NRGBA{
			R: uint8(min(255, int(fill.R)+50)),
			G: uint8(min(255, int(fill.G)+50)),
			B: uint8(min(255, int(fill.B)+50)),
			A: fill.A,
		}
		fillRect(screen, barX, barY, barW, max(1, barH/3), highlight)
		fillRect(screen, barX, barY, barW, 1, color.NRGBA{R: 255, G: 255, B: 255, A: 48})
		dark := color.NRGBA{
			R: uint8(max(0, int(fill.R)-40)),
			G: uint8(max(0, int(fill.G)-40)),
			B: uint8(max(0, int(fill.B)-40)),
			A: fill.A,
		}
		fillRect(screen, barX, barY+barH-1, barW, 1, dark)
	}

	DrawBorder(screen, rect, theme.BorderDark, theme.BorderMid, theme.BorderLight)
	DrawTextCentered(screen, label, rect.Add(image.Pt(1, 1)), color.NRGBA{A: 180})
	DrawTextCentered(screen, label, rect, theme.Text)
}

func DrawText(screen *ebiten.Image, label string, x, y int, clr color.Color) {
	op := &textv2.DrawOptions{}
	op.GeoM.Translate(float64(x), float64(y)-float64(Font().Metrics().Ascent.Ceil()))
	op.ColorScale.ScaleWithColor(clr)
	textv2.Draw(screen, label, textFace, op)
}

func DrawTextf(screen *ebiten.Image, x, y int, clr color.Color, format string, args ...any) {
	DrawText(screen, fmt.Sprintf(format, args...), x, y, clr)
}

func DrawTextCentered(screen *ebiten.Image, label string, rect image.Rectangle, clr color.Color) {
	x := rect.Min.X + (rect.Dx()-MeasureText(label))/2
	y := rect.Min.Y + (rect.Dy()+Font().Metrics().Ascent.Ceil())/2 - 2
	DrawText(screen, label, x, y, clr)
}

func DrawTextWrappedCentered(screen *ebiten.Image, label string, rect image.Rectangle, clr color.Color) {
	lines := WrapText(label, rect.Dx())
	if len(lines) == 0 {
		return
	}

	lineHeight := Font().Metrics().Height.Ceil()
	totalHeight := len(lines) * lineHeight
	startY := rect.Min.Y + (rect.Dy()-totalHeight)/2 + Font().Metrics().Ascent.Ceil() - 1
	for i, line := range lines {
		x := rect.Min.X + (rect.Dx()-MeasureText(line))/2
		DrawText(screen, line, x, startY+i*lineHeight, clr)
	}
}

func WrapText(label string, maxWidth int) []string {
	label = strings.TrimSpace(label)
	if label == "" || maxWidth <= 0 {
		return nil
	}

	words := strings.Fields(label)
	if len(words) == 0 {
		return nil
	}

	lines := make([]string, 0, 4)
	current := words[0]
	for _, word := range words[1:] {
		candidate := current + " " + word
		if MeasureText(candidate) <= maxWidth {
			current = candidate
			continue
		}

		lines = append(lines, current)
		current = word

		for MeasureText(current) > maxWidth && len(current) > 1 {
			cut := max(1, maxWidth/7-1)
			if cut >= len(current) {
				break
			}
			lines = append(lines, current[:cut]+"-")
			current = current[cut:]
		}
	}

	lines = append(lines, current)
	return lines
}

func MeasureText(label string) int {
	w, _ := textv2.Measure(label, textFace, 0)
	return int(w)
}

func DrawBorder(screen *ebiten.Image, rect image.Rectangle, shadow, mid, highlight color.Color) {
	fillRect(screen, float64(rect.Min.X), float64(rect.Min.Y), float64(rect.Dx()), 1, highlight)
	fillRect(screen, float64(rect.Min.X), float64(rect.Min.Y), 1, float64(rect.Dy()), highlight)
	fillRect(screen, float64(rect.Min.X), float64(rect.Max.Y-1), float64(rect.Dx()), 1, shadow)
	fillRect(screen, float64(rect.Max.X-1), float64(rect.Min.Y), 1, float64(rect.Dy()), shadow)

	fillRect(screen, float64(rect.Min.X+1), float64(rect.Min.Y+1), float64(rect.Dx()-2), 1, mid)
	fillRect(screen, float64(rect.Min.X+1), float64(rect.Min.Y+1), 1, float64(rect.Dy()-2), mid)
	fillRect(screen, float64(rect.Min.X+1), float64(rect.Max.Y-2), float64(rect.Dx()-2), 1, shadow)
	fillRect(screen, float64(rect.Max.X-2), float64(rect.Min.Y+1), 1, float64(rect.Dy()-2), shadow)
}

func drawCornerAccents(screen *ebiten.Image, rect image.Rectangle, accent color.NRGBA) {
	cs := float64(8)
	x0, y0 := float64(rect.Min.X), float64(rect.Min.Y)
	x1, y1 := float64(rect.Max.X), float64(rect.Max.Y)
	fillRect(screen, x0, y0, cs, 1, accent)
	fillRect(screen, x0, y0, 1, cs, accent)
	fillRect(screen, x1-cs, y0, cs, 1, accent)
	fillRect(screen, x1-1, y0, 1, cs, accent)
	fillRect(screen, x0, y1-1, cs, 1, accent)
	fillRect(screen, x0, y1-cs, 1, cs, accent)
	fillRect(screen, x1-cs, y1-1, cs, 1, accent)
	fillRect(screen, x1-1, y1-cs, 1, cs, accent)
}

func DrawInnerShade(screen *ebiten.Image, rect image.Rectangle, theme Theme) {
	fillRect(screen, float64(rect.Min.X), float64(rect.Min.Y), float64(rect.Dx()), 2, color.NRGBA{R: 255, G: 255, B: 255, A: 12})
	fillRect(screen, float64(rect.Min.X), float64(rect.Max.Y-2), float64(rect.Dx()), 2, color.NRGBA{R: 0, G: 0, B: 0, A: 28})
	_ = theme
}

func blend(a, b color.NRGBA, t float64) color.NRGBA {
	return color.NRGBA{
		R: uint8(float64(a.R) + (float64(b.R)-float64(a.R))*t),
		G: uint8(float64(a.G) + (float64(b.G)-float64(a.G))*t),
		B: uint8(float64(a.B) + (float64(b.B)-float64(a.B))*t),
		A: uint8(float64(a.A) + (float64(b.A)-float64(a.A))*t),
	}
}

func ternary(ok bool, left, right color.NRGBA) color.NRGBA {
	if ok {
		return left
	}
	return right
}
