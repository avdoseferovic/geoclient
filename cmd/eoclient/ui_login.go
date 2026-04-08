package main

import (
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/avdo/eoweb/internal/game"
	clientui "github.com/avdo/eoweb/internal/ui"
	"github.com/avdo/eoweb/internal/ui/login"
)

func (g *Game) updateLogin() {
	if g.overlay.loginSubmitting {
		return
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		layout := login.LayoutFor(g.screenW, g.screenH)
		mx, my := ebiten.CursorPosition()
		newMode, newFocus, action := login.HandleClick(layout, mx, my, g.overlay.authMode)
		if action != "" {
			if action == "tab" {
				g.overlay.authMode = newMode
				if newFocus >= 0 {
					g.overlay.loginFocus = newFocus
				}
				g.connectError = ""
			} else if action == "focus" && newFocus >= 0 {
				g.overlay.loginFocus = newFocus
			} else if action == "submit" {
				switch g.overlay.authMode {
				case login.ModeCreateAccount:
					g.submitAccountCreate()
				case login.ModeSignIn:
					g.submitLogin()
				}
			}
			return
		}
	}

	fieldCount := 3
	if g.overlay.authMode == login.ModeCreateAccount {
		fieldCount = 6
	}
	if g.overlay.authMode == login.ModeCredits || g.overlay.authMode == login.ModeHome {
		fieldCount = 0
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyF2) {
		g.overlay.authMode = login.AuthMode((int(g.overlay.authMode) + 1) % 4)
		g.overlay.loginFocus = 0
		g.connectError = ""
	}

	if fieldCount > 0 && (inpututil.IsKeyJustPressed(ebiten.KeyTab) || inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) || inpututil.IsKeyJustPressed(ebiten.KeyArrowDown)) {
		g.overlay.loginFocus = (g.overlay.loginFocus + 1) % fieldCount
	}

	field := &g.overlay.loginUsername
	if g.overlay.authMode != login.ModeCredits && g.overlay.authMode != login.ModeHome {
		switch g.overlay.loginFocus {
		case login.FocusPassword:
			field = &g.overlay.loginPassword
		case login.FocusPasswordConfirm:
			field = &g.overlay.loginPassword2
		case login.FocusEmail:
			field = &g.overlay.loginEmail
		case login.FocusAddress:
			field = &g.overlay.loginAddress
		case login.FocusServer:
			field = &g.overlay.loginServerAddr
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) && len(*field) > 0 {
			*field = (*field)[:len(*field)-1]
		}
		for _, r := range ebiten.AppendInputChars(nil) {
			if r < 32 || r > 126 || len(*field) >= 24 {
				continue
			}
			*field = append(*field, r)
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		switch g.overlay.authMode {
		case login.ModeCreateAccount:
			g.submitAccountCreate()
		case login.ModeSignIn:
			g.submitLogin()
		}
	}
}

func (g *Game) submitLogin() {
	serverChanged, ok := g.applyServerAddressInput()
	if !ok {
		return
	}
	if serverChanged && g.client.GetState() == game.StateConnected {
		g.client.Disconnect()
		g.connected = false
		g.connectArmed = true
		g.connectError = ""
		g.overlay.statusMessage = "Server updated. Reconnecting..."
		return
	}
	username := strings.TrimSpace(string(g.overlay.loginUsername))
	password := string(g.overlay.loginPassword)
	if username == "" || password == "" {
		g.connectError = "Enter both account name and password"
		return
	}
	g.client.Username = username
	g.client.Password = password
	g.overlay.loginSubmitting = true
	g.connectError = ""
	g.overlay.statusMessage = "Sending credentials..."
	g.sendLogin()
}

func (g *Game) submitAccountCreate() {
	serverChanged, ok := g.applyServerAddressInput()
	if !ok {
		return
	}
	if serverChanged && g.client.GetState() == game.StateConnected {
		g.client.Disconnect()
		g.connected = false
		g.connectArmed = true
		g.connectError = ""
		g.overlay.statusMessage = "Server updated. Reconnecting..."
		return
	}
	username := strings.TrimSpace(string(g.overlay.loginUsername))
	password := string(g.overlay.loginPassword)
	confirm := string(g.overlay.loginPassword2)
	email := strings.TrimSpace(string(g.overlay.loginEmail))
	address := strings.TrimSpace(string(g.overlay.loginAddress))
	if errMsg := login.ValidateAccountCreate(username, password, confirm, email, address); errMsg != "" {
		g.connectError = errMsg
		return
	}
	g.client.Username = username
	g.client.Password = password
	g.client.PendingAccountCreate = &game.AccountCreateProfile{
		FullName: strings.TrimSpace(username),
		Location: address,
		Email:    email,
	}
	g.overlay.loginSubmitting = true
	g.connectError = ""
	g.overlay.statusMessage = "Requesting account creation..."
	g.sendAccountCreateRequest()
}

func (g *Game) drawLoginDialog(screen *ebiten.Image, theme clientui.Theme) {
	layout := login.LayoutFor(g.screenW, g.screenH)
	login.Draw(screen, theme, layout, g.overlay.authMode, g.overlay.loginFocus, g.overlay.ticks, g.overlay.loginUsername, g.overlay.loginPassword, g.overlay.loginPassword2, g.overlay.loginEmail, g.overlay.loginAddress, g.overlay.loginServerAddr, g.overlay.loginSubmitting, g.connectError)
}

func (g *Game) applyServerAddressInput() (bool, bool) {
	addr := strings.TrimSpace(string(g.overlay.loginServerAddr))
	if addr == "" {
		g.connectError = "Enter a server address"
		return false, false
	}
	changed := addr != g.serverAddr
	g.serverAddr = addr
	if g.serverConfigKey != "" {
		if err := saveServerPreference(g.serverConfigKey, addr); err != nil {
			g.connectError = "Unable to save server address"
			return false, false
		}
	}
	return changed, true
}
