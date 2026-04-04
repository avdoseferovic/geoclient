package overlay

import (
	"fmt"
	"image"
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	clientui "github.com/avdo/eoweb/internal/ui"
)

func PointInRect(x, y int, rect image.Rectangle) bool {
	return x >= rect.Min.X && x < rect.Max.X && y >= rect.Min.Y && y < rect.Max.Y
}

func CenteredRect(width, height, sw, sh int) image.Rectangle {
	x := (sw - width) / 2
	y := (sh - height) / 2
	return image.Rect(x, y, x+width, y+height)
}

func ClampInt(value, low, high int) int {
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}

func Colorize(base color.NRGBA, alpha uint8) color.NRGBA {
	base.A = alpha
	return base
}

func TernaryString(ok bool, left, right string) string {
	if ok {
		return left
	}
	return right
}

func TernaryColor(ok bool, left, right color.Color) color.Color {
	if ok {
		return left
	}
	return right
}

func FallbackString(value, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
}

func TruncateLabel(value string, maxLen int) string {
	value = strings.TrimSpace(value)
	if maxLen <= 0 || len(value) <= maxLen {
		return value
	}
	if maxLen <= 3 {
		return value[:maxLen]
	}
	return value[:maxLen-3] + "..."
}

func DotPulse(prefix string, ticks int) string {
	return prefix + strings.Repeat(".", (ticks/20)%4)
}

func DrawPulseBar(screen *ebiten.Image, rect image.Rectangle, theme clientui.Theme, ticks int) {
	clientui.DrawInset(screen, rect, theme, false)
	segmentW := 12
	for i := range 5 {
		alpha := uint8(50)
		if i == (ticks/12)%5 {
			alpha = 180
		}
		fill := Colorize(theme.Accent, alpha)
		r := image.Rect(rect.Min.X+6+i*(segmentW+6), rect.Min.Y+4, rect.Min.X+6+i*(segmentW+6)+segmentW, rect.Max.Y-4)
		clientui.FillRect(screen, float64(r.Min.X), float64(r.Min.Y), float64(r.Dx()), float64(r.Dy()), fill)
	}
}

func StatRatio(value, maxValue int) float64 {
	if maxValue <= 0 {
		return 1
	}
	if value <= 0 {
		return 0
	}
	return float64(value) / float64(maxValue)
}

func ExpForLevel(level int) int {
	if level <= 0 {
		return 0
	}
	return int(float64(level*level*level) * 133.1)
}

func TnlProgress(level, experience int) (remaining int, rangeValue int) {
	currentLevelExp := ExpForLevel(level)
	nextLevelExp := ExpForLevel(level + 1)
	if nextLevelExp <= currentLevelExp {
		return 0, 1
	}
	progressRange := nextLevelExp - currentLevelExp
	remaining = min(max(nextLevelExp-experience, 0), progressRange)
	return remaining, progressRange
}

func PaperdollPair(values []int) string {
	left, right := 0, 0
	if len(values) > 0 {
		left = values[0]
	}
	if len(values) > 1 {
		right = values[1]
	}
	return fmt.Sprintf("%d/%d", left, right)
}

func PaperdollValue(values []int, index int) int {
	if index < 0 || index >= len(values) {
		return 0
	}
	return values[index]
}

func BlendColors(a, b color.NRGBA, t float64) color.NRGBA {
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	return color.NRGBA{
		R: uint8(float64(a.R) + (float64(b.R)-float64(a.R))*t),
		G: uint8(float64(a.G) + (float64(b.G)-float64(a.G))*t),
		B: uint8(float64(a.B) + (float64(b.B)-float64(a.B))*t),
		A: uint8(float64(a.A) + (float64(b.A)-float64(a.A))*t),
	}
}
