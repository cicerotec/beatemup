package main

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// Combo de 3 golpes: jab, jab, gancho (derruba).
var playerCombo = []*AttackDef{
	{Name: "jab", Startup: 3, Active: 3, Recovery: 9, Reach: 26, Damage: 3, Push: 2},
	{Name: "jab2", Startup: 3, Active: 3, Recovery: 10, Reach: 26, Damage: 3, Push: 2},
	{Name: "gancho", Startup: 5, Active: 4, Recovery: 18, Reach: 30, Damage: 8, Knockdown: true},
}

var jumpKick = &AttackDef{Name: "voadora", Startup: 3, Active: 22, Recovery: 0, Reach: 28, Damage: 7, Knockdown: true}

type Player struct {
	*Fighter
	comboIdx    int  // próximo golpe do combo
	comboWindow int  // frames para emendar o próximo golpe
	lastHit     bool // o último golpe acertou? (combo só continua se acertar, como no Final Fight)
	buffered    bool // apertou soco durante o golpe atual
}

func NewPlayer() *Player {
	return &Player{Fighter: &Fighter{
		Name: "PLAYER 1", X: 60, Y: 190, Facing: 1,
		HP: 100, MaxHP: 100, W: 20, H: 42, Poise: 2,
		Color: color.RGBA{60, 120, 230, 255},
	}}
}

func keyDown(keys ...ebiten.Key) bool {
	for _, k := range keys {
		if ebiten.IsKeyPressed(k) {
			return true
		}
	}
	return false
}

func keyJust(keys ...ebiten.Key) bool {
	for _, k := range keys {
		if inpututil.IsKeyJustPressed(k) {
			return true
		}
	}
	return false
}

func (p *Player) startCombo() {
	idx := 0
	if p.comboWindow > 0 && p.lastHit {
		idx = p.comboIdx
	}
	p.startAttack(playerCombo[idx])
	p.comboIdx = (idx + 1) % len(playerCombo)
	p.lastHit = false
	p.buffered = false
	p.comboWindow = 0
}

func (p *Player) update(g *Game) {
	f := p.Fighter
	atk := keyJust(ebiten.KeyZ, ebiten.KeyJ)
	jump := keyJust(ebiten.KeyX, ebiten.KeyK, ebiten.KeySpace)

	var dx, dy float64
	if keyDown(ebiten.KeyArrowLeft, ebiten.KeyA) {
		dx--
	}
	if keyDown(ebiten.KeyArrowRight, ebiten.KeyD) {
		dx++
	}
	if keyDown(ebiten.KeyArrowUp, ebiten.KeyW) {
		dy--
	}
	if keyDown(ebiten.KeyArrowDown, ebiten.KeyS) {
		dy++
	}

	if p.comboWindow > 0 {
		p.comboWindow--
	}

	switch f.State {
	case StIdle, StWalk:
		switch {
		case atk:
			p.startCombo()
		case jump:
			f.VZ = 6.2
			f.VX = dx * 1.7
			f.setState(StJump)
		default:
			f.X += dx * 1.7
			f.Y += dy * 1.1
			if dx != 0 {
				f.Facing = dx
			}
			if dx != 0 || dy != 0 {
				if f.State != StWalk {
					f.setState(StWalk)
				}
			} else if f.State != StIdle {
				f.setState(StIdle)
			}
		}
	case StJump:
		if atk && f.Atk == nil {
			f.startAttack(jumpKick)
		}
	case StAttack:
		if atk {
			p.buffered = true
		}
		// Cancelamento: se acertou e já passou dos frames ativos, emenda o próximo golpe.
		if p.buffered && p.lastHit && p.comboIdx != 0 && f.AtkTimer >= f.Atk.Startup+f.Atk.Active {
			p.comboWindow = 1
			p.startCombo()
		}
	}

	wasAttacking := f.State == StAttack
	f.tick()
	if wasAttacking && f.State != StAttack {
		p.comboWindow = 20 // janela curta para emendar
	}

	// Limites: a rua (Y) e a câmera (X) — no Final Fight não dá pra voltar.
	f.Y = clamp(f.Y, streetTop, streetBottom)
	f.X = clamp(f.X, g.camX+12, g.camX+screenW-12)
}
