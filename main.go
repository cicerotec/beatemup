// Protótipo de beat 'em up estilo Final Fight em Go + Ebitengine.
//
// Rodar:  go mod tidy && go run .
// Controles: setas/WASD = mover, Z/J = soco (combo), X/K/Espaço = pulo (soco no ar = voadora)
package main

import (
	"log"
	"math/rand"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	screenW      = 320 // resolução "de arcade"; a janela amplia 3x
	screenH      = 240
	streetTop    = 150.0
	streetBottom = 230.0
	levelWidth   = 1700.0
)

// Wave = uma "onda" de inimigos. A câmera trava até todos caírem.
type Spawn struct {
	Kind      EnemyKind
	FromRight bool
	Y         float64
}

type Wave struct {
	CamX   float64
	Spawns []Spawn
}

var waves = []Wave{
	{0, []Spawn{{Punk, true, 170}, {Punk, true, 215}}},
	{420, []Spawn{{Punk, true, 160}, {Punk, false, 200}, {Punk, true, 225}}},
	{900, []Spawn{{Brute, true, 190}, {Punk, false, 165}, {Punk, true, 220}}},
	{1380, []Spawn{{Brute, true, 170}, {Brute, false, 215}, {Punk, true, 195}, {Punk, false, 160}}},
}

type Game struct {
	player     *Player
	enemies    []*Enemy
	camX       float64
	waveIdx    int
	waveActive bool
	hitstop    int // congela o jogo alguns frames no impacto
	shake      int
	frame      int
	score      int
	lastHit    *Enemy // barra de vida do último inimigo atingido
}

func NewGame() *Game {
	return &Game{player: NewPlayer()}
}

func (g *Game) Update() error {
	g.frame++
	if g.shake > 0 {
		g.shake--
	}
	if g.hitstop > 0 {
		g.hitstop--
		return nil
	}

	p := g.player
	if p.State == StDead && keyJust(ebiten.KeyEnter) {
		*g = *NewGame()
		return nil
	}

	g.updateCamera()
	p.update(g)
	g.assignTokens()
	for _, e := range g.enemies {
		e.update(g)
	}
	g.separateEnemies()
	g.resolveHits()

	// Remove inimigos que já sumiram.
	alive := g.enemies[:0]
	for _, e := range g.enemies {
		if !e.Removed {
			alive = append(alive, e)
		}
	}
	g.enemies = alive
	return nil
}

func (g *Game) updateCamera() {
	if g.waveActive {
		if len(g.enemies) == 0 {
			g.waveActive = false
			g.waveIdx++
		}
		return
	}
	// Segue o jogador, mas nunca volta (igual ao Final Fight).
	target := clamp(g.player.X-screenW*0.45, 0, levelWidth-screenW)
	if target > g.camX {
		g.camX += (target - g.camX) * 0.15
	}
	if g.waveIdx < len(waves) && g.camX >= waves[g.waveIdx].CamX-0.5 {
		w := waves[g.waveIdx]
		g.camX = w.CamX
		g.waveActive = true
		for _, s := range w.Spawns {
			x := g.camX - 30
			if s.FromRight {
				x = g.camX + screenW + 30
			}
			g.enemies = append(g.enemies, NewEnemy(s.Kind, x, s.Y))
		}
	}
}

// assignTokens: no máximo 2 inimigos atacam ao mesmo tempo; os outros rondam.
func (g *Game) assignTokens() {
	const maxAttackers = 2
	n := 0
	var candidates []*Enemy
	for _, e := range g.enemies {
		if e.HasToken {
			n++
		} else if e.Cooldown == 0 && (e.State == StIdle || e.State == StWalk) {
			candidates = append(candidates, e)
		}
	}
	px := g.player.X
	sort.Slice(candidates, func(i, j int) bool {
		return abs(candidates[i].X-px) < abs(candidates[j].X-px)
	})
	for _, e := range candidates {
		if n >= maxAttackers {
			break
		}
		e.HasToken = true
		n++
	}
}

// Evita que inimigos fiquem empilhados no mesmo pixel.
func (g *Game) separateEnemies() {
	for i, a := range g.enemies {
		for _, b := range g.enemies[i+1:] {
			if abs(a.X-b.X) < 16 && abs(a.Y-b.Y) < 6 {
				d := sign(a.X-b.X) * 0.5
				if a.X == b.X {
					d = rand.Float64() - 0.5
				}
				a.X += d
				b.X -= d
			}
		}
	}
}

func (g *Game) resolveHits() {
	p := g.player
	for _, e := range g.enemies {
		if hit, ko := tryHit(p.Fighter, e.Fighter); hit {
			p.lastHit = true
			g.lastHit = e
			g.score += p.Atk.Damage * 10
			if !e.alive() {
				g.score += e.Score
			}
			g.hitstop, g.shake = 4, 3
			if ko {
				g.hitstop, g.shake = 7, 8
			}
		}
	}
	for _, e := range g.enemies {
		if hit, ko := tryHit(e.Fighter, p.Fighter); hit {
			g.hitstop, g.shake = 5, 5
			if ko {
				g.hitstop, g.shake = 8, 10
			}
		}
	}
}

func (g *Game) Layout(_, _ int) (int, int) { return screenW, screenH }

func main() {
	ebiten.SetWindowSize(screenW*3, screenH*3)
	ebiten.SetWindowTitle("Beat 'em up — protótipo")
	if err := ebiten.RunGame(NewGame()); err != nil {
		log.Fatal(err)
	}
}
