package login

import (
	"image"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	clientui "github.com/avdo/eoweb/internal/ui"
	"github.com/avdo/eoweb/internal/ui/overlay"
)

type AuthMode int

const (
	ModeHome AuthMode = iota
	ModeSignIn
	ModeCreateAccount
	ModeCredits
)

const (
	FocusUsername = iota
	FocusPassword
	FocusPasswordConfirm
	FocusEmail
	FocusAddress
)

type Layout struct {
	Dialog      image.Rectangle
	MenuRect    image.Rectangle
	ContentRect image.Rectangle
	SignInTab   image.Rectangle
	CreateTab   image.Rectangle
	CreditsTab  image.Rectangle
	UserRect    image.Rectangle
	PassRect    image.Rectangle
	ConfirmRect image.Rectangle
	EmailRect   image.Rectangle
	AddressRect image.Rectangle
	SubmitRect  image.Rectangle
}

func LayoutFor(sw, sh int) Layout {
	dialog := overlay.CenteredRect(560, 452, sw, sh)
	menuRect := image.Rect(dialog.Min.X+18, dialog.Min.Y+42, dialog.Min.X+182, dialog.Max.Y-18)
	contentRect := image.Rect(menuRect.Max.X+14, dialog.Min.Y+42, dialog.Max.X-18, dialog.Max.Y-18)
	contentLeft := contentRect.Min.X + 20
	contentRight := contentRect.Max.X - 20
	const inputHeight = 28
	const rowStep = 56
	const submitHeight = 26
	const bottomPad = 22
	firstRowTop := contentRect.Max.Y - bottomPad - submitHeight - 18 - inputHeight - rowStep*4
	userRect := image.Rect(contentLeft, firstRowTop, contentRight, firstRowTop+inputHeight)
	passRect := userRect.Add(image.Pt(0, rowStep))
	confirmRect := passRect.Add(image.Pt(0, rowStep))
	emailRect := confirmRect.Add(image.Pt(0, rowStep))
	addressRect := emailRect.Add(image.Pt(0, rowStep))
	submitRect := image.Rect(contentRight-160, contentRect.Max.Y-bottomPad-submitHeight, contentRight, contentRect.Max.Y-bottomPad)
	const menuButtonHeight = 26
	const menuButtonGap = 8
	menuButtonsBottom := menuRect.Max.Y - 18
	creditsTab := image.Rect(menuRect.Min.X+12, menuButtonsBottom-menuButtonHeight, menuRect.Max.X-12, menuButtonsBottom)
	createTab := creditsTab.Add(image.Pt(0, -(menuButtonHeight + menuButtonGap)))
	signInTab := createTab.Add(image.Pt(0, -(menuButtonHeight + menuButtonGap)))
	return Layout{
		Dialog:      dialog,
		MenuRect:    menuRect,
		ContentRect: contentRect,
		SignInTab:   signInTab,
		CreateTab:   createTab,
		CreditsTab:  creditsTab,
		UserRect:    userRect,
		PassRect:    passRect,
		ConfirmRect: confirmRect,
		EmailRect:   emailRect,
		AddressRect: addressRect,
		SubmitRect:  submitRect,
	}
}

func Draw(screen *ebiten.Image, theme clientui.Theme, layout Layout, mode AuthMode, focus int, ticks int, username, password, password2, email, address []rune, submitting bool, connectError string) {
	rect := layout.Dialog
	clientui.DrawPanel(screen, rect, theme, clientui.PanelOptions{Accent: theme.Accent})
	clientui.DrawInset(screen, layout.MenuRect, theme, false)
	clientui.DrawInset(screen, layout.ContentRect, theme, false)

	clientui.DrawButton(screen, layout.SignInTab, theme, "Sign In", mode == ModeSignIn, false)
	clientui.DrawButton(screen, layout.CreateTab, theme, "Create Account", mode == ModeCreateAccount, false)
	clientui.DrawButton(screen, layout.CreditsTab, theme, "Credits", mode == ModeCredits, false)

	switch mode {
	case ModeHome:
		drawLandingPanel(screen, theme, layout)
	case ModeCredits:
		drawCreditsPanel(screen, theme, layout)
	default:
		drawAuthPanel(screen, theme, layout, mode, focus, ticks, username, password, password2, email, address, submitting)
	}

	statusColor := theme.TextDim
	status := ""
	if submitting {
		status = overlay.TernaryString(mode == ModeCreateAccount, "Awaiting server approval...", "Awaiting server response...")
	}
	if connectError != "" {
		status = connectError
		statusColor = theme.Danger
	}
	if status != "" {
		clientui.DrawTextWrappedCentered(screen, status, image.Rect(layout.ContentRect.Min.X+16, layout.ContentRect.Max.Y-34, layout.ContentRect.Max.X-16, layout.ContentRect.Max.Y-12), statusColor)
	}
}

func drawLandingPanel(screen *ebiten.Image, theme clientui.Theme, layout Layout) {
	clientui.DrawInset(screen, image.Rect(layout.ContentRect.Min.X+20, layout.ContentRect.Min.Y+20, layout.ContentRect.Max.X-20, layout.ContentRect.Max.Y-20), theme, false)
}

func drawAuthPanel(screen *ebiten.Image, theme clientui.Theme, layout Layout, mode AuthMode, focus, ticks int, username, password, password2, email, address []rune, submitting bool) {
	clientui.DrawText(screen, "Account", layout.UserRect.Min.X, layout.UserRect.Min.Y-8, theme.TextDim)
	clientui.DrawInset(screen, layout.UserRect, theme, focus == FocusUsername)
	userText := string(username)
	if focus == FocusUsername && ticks%40 < 20 {
		userText += "_"
	}
	clientui.DrawText(screen, userText, layout.UserRect.Min.X+10, layout.UserRect.Min.Y+18, theme.Text)

	clientui.DrawText(screen, "Password", layout.PassRect.Min.X, layout.PassRect.Min.Y-8, theme.TextDim)
	clientui.DrawInset(screen, layout.PassRect, theme, focus == FocusPassword)
	passText := strings.Repeat("*", len(password))
	if focus == FocusPassword && ticks%40 < 20 {
		passText += "_"
	}
	clientui.DrawText(screen, passText, layout.PassRect.Min.X+10, layout.PassRect.Min.Y+18, theme.Text)

	if mode == ModeCreateAccount {
		clientui.DrawText(screen, "Confirm Password", layout.ConfirmRect.Min.X, layout.ConfirmRect.Min.Y-8, theme.TextDim)
		clientui.DrawInset(screen, layout.ConfirmRect, theme, focus == FocusPasswordConfirm)
		confirmText := strings.Repeat("*", len(password2))
		if focus == FocusPasswordConfirm && ticks%40 < 20 {
			confirmText += "_"
		}
		clientui.DrawText(screen, confirmText, layout.ConfirmRect.Min.X+10, layout.ConfirmRect.Min.Y+18, theme.Text)

		clientui.DrawText(screen, "Email", layout.EmailRect.Min.X, layout.EmailRect.Min.Y-8, theme.TextDim)
		clientui.DrawInset(screen, layout.EmailRect, theme, focus == FocusEmail)
		emailText := string(email)
		if focus == FocusEmail && ticks%40 < 20 {
			emailText += "_"
		}
		clientui.DrawText(screen, emailText, layout.EmailRect.Min.X+10, layout.EmailRect.Min.Y+18, theme.Text)

		clientui.DrawText(screen, "Address", layout.AddressRect.Min.X, layout.AddressRect.Min.Y-8, theme.TextDim)
		clientui.DrawInset(screen, layout.AddressRect, theme, focus == FocusAddress)
		addressText := string(address)
		if focus == FocusAddress && ticks%40 < 20 {
			addressText += "_"
		}
		clientui.DrawText(screen, addressText, layout.AddressRect.Min.X+10, layout.AddressRect.Min.Y+18, theme.Text)
	}

	buttonLabel := "Enter Realm"
	if mode == ModeCreateAccount {
		buttonLabel = "Create Account"
	}
	if submitting {
		buttonLabel = overlay.TernaryString(mode == ModeCreateAccount, "Creating", "Signing In")
	}
	clientui.DrawButton(screen, layout.SubmitRect, theme, buttonLabel, true, submitting)
}

func drawCreditsPanel(screen *ebiten.Image, theme clientui.Theme, layout Layout) {
	creditsRect := image.Rect(layout.ContentRect.Min.X+20, layout.ContentRect.Min.Y+54, layout.ContentRect.Max.X-20, layout.ContentRect.Max.Y-44)
	clientui.DrawInset(screen, creditsRect, theme, false)
	clientui.DrawText(screen, "Credits", creditsRect.Min.X+14, creditsRect.Min.Y+18, theme.Text)
	clientui.DrawTextWrappedCentered(screen, "Endless Online by vult-r", image.Rect(creditsRect.Min.X+12, creditsRect.Min.Y+42, creditsRect.Max.X-12, creditsRect.Min.Y+72), theme.TextDim)
	clientui.DrawTextWrappedCentered(screen, "GeoClient by avdoseferovic", image.Rect(creditsRect.Min.X+12, creditsRect.Min.Y+74, creditsRect.Max.X-12, creditsRect.Min.Y+104), theme.TextDim)
	clientui.DrawTextWrappedCentered(screen, "endless-online.com", image.Rect(creditsRect.Min.X+12, creditsRect.Min.Y+120, creditsRect.Max.X-12, creditsRect.Min.Y+150), theme.Accent)
	clientui.DrawTextWrappedCentered(screen, "Use the left rail to return to Sign In or Create Account.", image.Rect(creditsRect.Min.X+18, creditsRect.Max.Y-52, creditsRect.Max.X-18, creditsRect.Max.Y-20), theme.TextDim)
}

func DrawConnecting(screen *ebiten.Image, theme clientui.Theme, sw, sh, ticks int, statusMessage, connectError string) {
	rect := overlay.CenteredRect(360, 128, sw, sh)
	clientui.DrawPanel(screen, rect, theme, clientui.PanelOptions{Title: "Gate Link", Accent: theme.Accent})
	status := "Contacting server"
	if connectError != "" {
		status = "Connection failed"
	}
	clientui.DrawTextCentered(screen, status, image.Rect(rect.Min.X+24, rect.Min.Y+26, rect.Max.X-24, rect.Min.Y+54), theme.Text)

	line := statusMessage
	if line == "" {
		line = overlay.DotPulse("Opening relay", ticks)
	}
	if connectError != "" {
		line = connectError
	}
	clientui.DrawTextCentered(screen, line, image.Rect(rect.Min.X+28, rect.Min.Y+56, rect.Max.X-28, rect.Min.Y+82), overlay.TernaryColor(connectError == "", theme.TextDim, theme.Danger))

	footer := "Please wait"
	if connectError != "" {
		footer = "Press Enter or R to try again"
	}
	clientui.DrawTextCentered(screen, footer, image.Rect(rect.Min.X+20, rect.Max.Y-34, rect.Max.X-20, rect.Max.Y-14), theme.TextDim)
	spinnerRect := image.Rect(rect.Min.X+rect.Dx()/2-44, rect.Min.Y+88, rect.Min.X+rect.Dx()/2+44, rect.Min.Y+104)
	overlay.DrawPulseBar(screen, spinnerRect, theme, ticks)
	clientui.DrawTextCentered(screen, "Endless Offline Native Client", image.Rect(0, sh-38, sw, sh-18), theme.TextDim)
}

func HandleClick(layout Layout, mx, my int, mode AuthMode) (newMode AuthMode, newFocus int, action string) {
	if !overlay.PointInRect(mx, my, layout.Dialog) {
		return mode, -1, ""
	}
	switch {
	case overlay.PointInRect(mx, my, layout.SignInTab):
		return ModeSignIn, FocusUsername, "tab"
	case overlay.PointInRect(mx, my, layout.CreateTab):
		return ModeCreateAccount, FocusUsername, "tab"
	case overlay.PointInRect(mx, my, layout.CreditsTab):
		return ModeCredits, -1, "tab"
	case overlay.PointInRect(mx, my, layout.UserRect):
		if mode == ModeCredits || mode == ModeHome {
			return mode, -1, "absorbed"
		}
		return mode, FocusUsername, "focus"
	case overlay.PointInRect(mx, my, layout.PassRect):
		if mode == ModeCredits || mode == ModeHome {
			return mode, -1, "absorbed"
		}
		return mode, FocusPassword, "focus"
	case mode == ModeCreateAccount && overlay.PointInRect(mx, my, layout.ConfirmRect):
		return mode, FocusPasswordConfirm, "focus"
	case mode == ModeCreateAccount && overlay.PointInRect(mx, my, layout.EmailRect):
		return mode, FocusEmail, "focus"
	case mode == ModeCreateAccount && overlay.PointInRect(mx, my, layout.AddressRect):
		return mode, FocusAddress, "focus"
	case overlay.PointInRect(mx, my, layout.SubmitRect):
		return mode, -1, "submit"
	default:
		return mode, -1, "absorbed"
	}
}

func ValidateAccountCreate(username, password, confirm, email, address string) string {
	if strings.TrimSpace(username) == "" || password == "" {
		return "Enter both account name and password"
	}
	if password != confirm {
		return "Passwords do not match"
	}
	if strings.TrimSpace(email) == "" {
		return "Enter an email address"
	}
	if strings.TrimSpace(address) == "" {
		return "Enter an address"
	}
	return ""
}

// DrawBackdrop draws the animated game backdrop for pre-game screens.
func DrawBackdrop(screen *ebiten.Image, theme clientui.Theme, ticks int) {
	clientui.DrawBackdrop(screen, theme, ticks)
}
