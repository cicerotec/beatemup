package main

import (
	"image/color"
	"math/rand"
)

type EnemyKind int

const (
	Punk  EnemyKind = iota // rápido e fraco
	Brute                  // lento, aguenta mais e derruba
)

type Enemy struct {
	*Fighter
	Kind     EnemyKind
	Speed    float64
	attack   *AttackDef
	HasToken bool // "ficha" de ataque: só 2 inimigos atacam por vez
	Cooldown int
	HoverY   float64
	Removed  bool
	Score    int
}

func NewEnemy(kind EnemyKind, x, y float64) *Enemy {
	e := &Enemy{Kind: kind, HoverY: y, Cooldown: 40 + rand.Intn(60)}
	switch kind {
	case Punk:
		e.Fighter = &Fighter{Name: "PUNK", W: 18, H: 38, HP: 18, Poise: 4,
			Color: color.RGBA{220, 70, 70, 255}}
		e.Speed = 0.95
		e.attack = &AttackDef{Name: "soco", Startup: 16, Active: 4, Recovery: 24, Reach: 24, Damage: 5, Push: 3}
		e.Score = 100
	case Brute:
		e.Fighter = &Fighter{Name: "BRUTE", W: 26, H: 48, HP: 40, Poise: 6,
			Color: color.RGBA{150, 90, 200, 255}}
		e.Speed = 0.6
		e.attack = &AttackDef{Name: "porrada", Startup: 24, Active: 5, Recovery: 28, Reach: 30, Damage: 10, Knockdown: true}
		e.Score = 300
	}
	e.MaxHP = e.HP
	e.X, e.Y, e.Facing = x, y, -1
	return e
}

// telegraphing = está preparando o golpe (o jogador tem tempo de reagir).
func (e *Enemy) telegraphing() bool {
	return e.State == StAttack && e.Atk != nil && e.AtkTimer < e.Atk.Startup
}

func (e *Enemy) update(g *Game) {
	f := e.Fighter
	p := g.player.Fighter
	if e.Cooldown > 0 {
		e.Cooldown--
	}

	switch f.State {
	case StIdle, StWalk:
		side := sign(f.X - p.X) // de que lado do jogador eu estou
		var tx, ty float64
		if e.HasToken && p.alive() {
			// Vai para a frente do jogador, na mesma linha.
			tx, ty = p.X+side*(e.attack.Reach-4), p.Y
		} else {
			// Fica rondando a uma distância segura.
			if f.Timer%90 == 0 {
				e.HoverY = streetTop + rand.Float64()*(streetBottom-streetTop)
			}
			tx, ty = p.X+side*(70+float64(e.Kind)*15), e.HoverY
		}
		dx, dy := tx-f.X, ty-f.Y
		moving := false
		if abs(dx) > 2 {
			f.X += sign(dx) * e.Speed
			moving = true
		}
		if abs(dy) > 1.5 {
			f.Y += sign(dy) * e.Speed * 0.7
			moving = true
		}
		f.Facing = -side
		if moving && f.State != StWalk {
			f.State = StWalk
		} else if !moving && f.State != StIdle {
			f.State = StIdle
		}
		if e.HasToken && abs(dx) < 5 && abs(f.Y-p.Y) < 6 && p.vulnerable() {
			f.startAttack(e.attack)
		}
	case StHurt, StKnockdown:
		if e.HasToken {
			e.HasToken = false
			e.Cooldown = 40
		}
	}

	wasAttacking := f.State == StAttack
	f.tick()
	if wasAttacking && f.State != StAttack {
		e.HasToken = false
		e.Cooldown = 50 + rand.Intn(70)
	}

	f.Y = clamp(f.Y, streetTop, streetBottom)
	if f.State == StDead && f.Timer > 40 {
		e.Removed = true
	}
}
