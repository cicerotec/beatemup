package main

import (
	"fmt"
	"image/color"
	"math"
	"math/rand"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

func rect(dst *ebiten.Image, x, y, w, h float64, c color.Color) {
	vector.DrawFilledRect(dst, float32(x), float32(y), float32(w), float32(h), c, false)
}

func shade(c color.RGBA, k float64) color.RGBA {
	m := func(v uint8) uint8 { return uint8(math.Min(255, float64(v)*k)) }
	return color.RGBA{m(c.R), m(c.G), m(c.B), c.A}
}

func (g *Game) Draw(screen *ebiten.Image) {
	// Tremida de tela: desloca todo o "mundo".
	var sx, sy float64
	if g.shake > 0 {
		sx, sy = float64(rand.Intn(5)-2), float64(rand.Intn(5)-2)
	}
	ox := -g.camX + sx

	g.drawBackground(screen, ox, sy)

	// Ordena pela profundidade (Y): quem está mais "embaixo" é desenhado por cima.
	type drawable struct {
		f     *Fighter
		enemy *Enemy
	}
	list := []drawable{{f: g.player.Fighter}}
	for _, e := range g.enemies {
		list = append(list, drawable{e.Fighter, e})
	}
	sort.SliceStable(list, func(i, j int) bool { return list[i].f.Y < list[j].f.Y })
	for _, d := range list {
		drawFighter(screen, d.f, d.enemy, ox, sy)
	}

	g.drawHUD(screen)
}

func (g *Game) drawBackground(dst *ebiten.Image, ox, oy float64) {
	dst.Fill(color.RGBA{36, 28, 58, 255})

	// Prédios ao fundo com parallax (andam mais devagar que a rua).
	par := ox * 0.5
	for i := 0; i < 40; i++ {
		bx := float64(i)*70 + par
		if bx < -80 || bx > screenW {
			continue
		}
		h := 60 + float64((i*37)%50)
		top := 132 - h + oy
		c := color.RGBA{uint8(50 + (i*13)%30), uint8(45 + (i*7)%20), 80, 255}
		rect(dst, bx, top, 62, h, c)
		for wy := top + 8; wy < 120+oy; wy += 14 {
			for wx := bx + 6; wx < bx+56; wx += 12 {
				if (int(wx)+int(wy)+i)%3 != 0 {
					rect(dst, wx, wy, 5, 7, color.RGBA{230, 200, 90, 255})
				}
			}
		}
	}

	// Calçada e rua.
	rect(dst, 0, 132+oy, screenW, 16, color.RGBA{120, 115, 125, 255})
	rect(dst, 0, 146+oy, screenW, 3, color.RGBA{90, 85, 95, 255})
	rect(dst, 0, 149+oy, screenW, 100, color.RGBA{64, 62, 74, 255})
	start := math.Mod(ox, 60)
	for x := start - 60; x < screenW; x += 60 {
		rect(dst, x, 196+oy, 28, 3, color.RGBA{200, 190, 120, 255})
	}
	// Placa de "fim de fase".
	endX := levelWidth - 40 + ox
	rect(dst, endX, 90+oy, 4, 60, color.RGBA{180, 180, 180, 255})
	rect(dst, endX-14, 86+oy, 32, 14, color.RGBA{40, 160, 70, 255})
}

func drawFighter(dst *ebiten.Image, f *Fighter, e *Enemy, ox, oy float64) {
	x := f.X + ox
	ground := f.Y + oy

	// Sombra no chão (fica no Y mesmo quando pula: é isso que vende a profundidade).
	rect(dst, x-f.W/2+1, ground-2, f.W-2, 4, color.RGBA{0, 0, 0, 90})

	if f.State == StDead && (f.Timer/4)%2 == 0 {
		return
	}
	if f.Invuln > 0 && (f.Invuln/3)%2 == 0 {
		return
	}

	base := f.Color
	if f.Flash > 0 {
		base = color.RGBA{255, 255, 255, 255}
	}
	dark := shade(base, 0.6)
	skin := color.RGBA{240, 190, 150, 255}
	if f.Flash > 0 {
		skin = base
	}
	feet := ground - f.Z

	// Caído ou voando de costas: desenha deitado.
	if f.State == StDown || f.State == StDead || (f.State == StKnockdown && f.Timer > 6) {
		lw, lh := f.H, f.W*0.6
		rect(dst, x-lw/2, feet-lh, lw, lh, base)
		hx := x - f.Facing*(lw/2) - f.Facing*2
		rect(dst, hx-5, feet-lh, 10, lh, skin)
		return
	}

	h := f.H
	if f.State == StGetup {
		h = f.H * 0.6
	}
	lean := 0.0
	if f.State == StHurt || f.State == StKnockdown {
		lean = -f.Facing * 3
	}
	top := feet - h
	head := f.W * 0.6

	rect(dst, x-f.W/2+lean, top+head, f.W, (h-head)*0.55, base) // tronco
	legY := top + head + (h-head)*0.55
	rect(dst, x-f.W/2, legY, f.W*0.42, feet-legY, dark) // pernas
	rect(dst, x+f.W*0.08, legY, f.W*0.42, feet-legY, dark)
	rect(dst, x-head/2+lean, top, head, head, skin) // cabeça
	eyeX := x + lean + f.Facing*head*0.2 - 1
	rect(dst, eyeX, top+head*0.35, 2, 2, color.RGBA{20, 20, 20, 255}) // olho = para onde olha

	// Braço/perna do golpe.
	if f.Atk != nil {
		armY := top + head + 4
		thick := 5.0
		if f.State == StJump { // voadora: perna esticada
			armY = legY + 2
			thick = 6
		}
		var length float64
		switch {
		case f.AtkTimer < f.Atk.Startup:
			length = 4 // recolhido (preparando)
		case f.isActive():
			length = f.Atk.Reach - 2
		default:
			length = f.Atk.Reach * 0.4
		}
		ax := x
		if f.Facing < 0 {
			ax = x - length
		}
		rect(dst, ax, armY, length, thick, skin)
	}

	// Aviso de ataque do inimigo: dá tempo do jogador reagir.
	if e != nil && e.telegraphing() {
		ebitenutil.DebugPrintAt(dst, "!", int(x)-3, int(top)-18)
	}
}

func bar(dst *ebiten.Image, x, y, w float64, hp, max int, c color.RGBA) {
	rect(dst, x-1, y-1, w+2, 8, color.RGBA{0, 0, 0, 255})
	rect(dst, x, y, w, 6, color.RGBA{140, 30, 30, 255})
	rect(dst, x, y, w*float64(hp)/float64(max), 6, c)
}

func (g *Game) drawHUD(dst *ebiten.Image) {
	p := g.player
	ebitenutil.DebugPrintAt(dst, p.Name, 8, 2)
	bar(dst, 8, 18, 110, p.HP, p.MaxHP, color.RGBA{250, 210, 40, 255})
	ebitenutil.DebugPrintAt(dst, fmt.Sprintf("SCORE %06d", g.score), 228, 2)

	if e := g.lastHit; e != nil && !e.Removed {
		ebitenutil.DebugPrintAt(dst, e.Name, 8, 26)
		bar(dst, 8, 42, 80, e.HP, e.MaxHP, color.RGBA{250, 140, 40, 255})
	}

	blink := (g.frame/20)%2 == 0
	switch {
	case p.State == StDead:
		ebitenutil.DebugPrintAt(dst, "GAME OVER", 130, 90)
		ebitenutil.DebugPrintAt(dst, "ENTER para recomecar", 98, 106)
	case g.waveIdx >= len(waves):
		ebitenutil.DebugPrintAt(dst, "STAGE CLEAR!", 122, 90)
	case !g.waveActive && blink:
		ebitenutil.DebugPrintAt(dst, "GO! -->", 262, 70)
	}

	if g.frame < 60*8 {
		ebitenutil.DebugPrintAt(dst, "Setas: mover  Z: soco  X: pulo", 70, 222)
	}
}
