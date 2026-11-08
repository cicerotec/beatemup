package main

import "image/color"

// State é o estado atual de um lutador (a "máquina de estados").
type State int

const (
	StIdle      State = iota // parado
	StWalk                   // andando
	StJump                   // no ar
	StAttack                 // golpe no chão
	StHurt                   // tomou dano (atordoado)
	StKnockdown              // voando depois de um golpe forte
	StDown                   // caído no chão
	StGetup                  // levantando (invulnerável)
	StDead                   // morto (pisca e some)
)

// AttackDef descreve um golpe em frames (o jogo roda a 60 frames por segundo).
//
//	Startup  = preparação (ainda não acerta)
//	Active   = frames em que o golpe acerta
//	Recovery = tempo vulnerável depois do golpe
type AttackDef struct {
	Name                      string
	Startup, Active, Recovery int
	Reach                     float64 // alcance à frente do corpo
	Damage                    int
	Knockdown                 bool    // derruba?
	Push                      float64 // quanto empurra o alvo
}

func (a *AttackDef) total() int { return a.Startup + a.Active + a.Recovery }

// Fighter é a base comum de jogador e inimigos.
//
// Coordenadas no estilo "belt scroll":
//
//	X = horizontal, Y = profundidade na rua (subir/descer), Z = altura do pulo.
type Fighter struct {
	Name           string
	X, Y, Z        float64
	VX, VZ         float64
	Facing         float64 // 1 = direita, -1 = esquerda
	State          State
	Timer          int // frames desde que entrou no estado atual
	HP, MaxHP      int
	W, H           float64
	Color          color.RGBA
	Flash, Invuln  int // frames de flash branco / invencibilidade
	Poise          int // quantos golpes seguidos aguenta antes de cair
	recentHits     int
	recentTimer    int
	Atk            *AttackDef
	AtkTimer       int
	hitSet         map[*Fighter]bool // quem já foi atingido por este golpe
}

const gravity = 0.35

func (f *Fighter) setState(s State) {
	f.State = s
	f.Timer = 0
}

func (f *Fighter) startAttack(a *AttackDef) {
	f.Atk = a
	f.AtkTimer = 0
	f.hitSet = map[*Fighter]bool{}
	if f.State != StJump {
		f.setState(StAttack)
	}
}

// isActive diz se o golpe atual está nos frames que acertam.
func (f *Fighter) isActive() bool {
	return f.Atk != nil && f.AtkTimer >= f.Atk.Startup && f.AtkTimer < f.Atk.Startup+f.Atk.Active
}

func (f *Fighter) vulnerable() bool {
	switch f.State {
	case StKnockdown, StDown, StGetup, StDead:
		return false
	}
	return f.Invuln == 0
}

func (f *Fighter) alive() bool { return f.HP > 0 }

// tick avança a física e as transições automáticas de estado. Chamado 1x por frame.
func (f *Fighter) tick() {
	if f.Atk != nil {
		f.AtkTimer++
		if f.AtkTimer >= f.Atk.total() {
			f.Atk = nil
			if f.State == StAttack {
				f.setState(StIdle)
			}
		}
	}

	switch f.State {
	case StJump:
		f.X += f.VX
		f.Z += f.VZ
		f.VZ -= gravity
		if f.Z <= 0 {
			f.Z = 0
			f.Atk = nil
			f.setState(StIdle)
		}
	case StHurt:
		if f.Timer > 14 {
			f.setState(StIdle)
		}
	case StKnockdown:
		f.X += f.VX
		f.Z += f.VZ
		f.VZ -= gravity
		if f.Z <= 0 {
			f.Z = 0
			f.VX = 0
			f.setState(StDown)
		}
	case StDown:
		if f.Timer > 40 {
			if f.HP <= 0 {
				f.setState(StDead)
			} else {
				f.setState(StGetup)
			}
		}
	case StGetup:
		if f.Timer > 16 {
			f.Invuln = 40
			f.setState(StIdle)
		}
	}

	if f.recentTimer > 0 {
		f.recentTimer--
		if f.recentTimer == 0 {
			f.recentHits = 0
		}
	}
	if f.Flash > 0 {
		f.Flash--
	}
	if f.Invuln > 0 {
		f.Invuln--
	}
	f.Timer++
}

// tryHit verifica se o golpe de "a" acerta "t". Retorna (acertou, derrubou).
func tryHit(a, t *Fighter) (bool, bool) {
	if !a.isActive() || a.hitSet[t] || !t.vulnerable() {
		return false, false
	}
	// Precisa estar quase na mesma linha de profundidade e altura.
	if abs(a.Y-t.Y) > 10 || abs(a.Z-t.Z) > 24 {
		return false, false
	}
	// Hitbox do golpe (à frente do atacante) x hurtbox do corpo do alvo.
	x0, x1 := a.X, a.X+a.Atk.Reach
	if a.Facing < 0 {
		x0, x1 = a.X-a.Atk.Reach, a.X
	}
	if x1 < t.X-t.W/2 || x0 > t.X+t.W/2 {
		return false, false
	}

	a.hitSet[t] = true
	t.HP -= a.Atk.Damage
	if t.HP < 0 {
		t.HP = 0
	}
	t.Flash = 6
	t.Atk = nil
	t.Facing = -a.Facing
	t.recentHits++
	t.recentTimer = 60

	if a.Atk.Knockdown || t.HP <= 0 || t.recentHits >= t.Poise || t.State == StJump {
		t.recentHits = 0
		t.VX = a.Facing * 2.2
		t.VZ = 4.5
		if t.Z < 1 {
			t.Z = 1
		}
		t.setState(StKnockdown)
		return true, true
	}
	t.X += a.Facing * a.Atk.Push
	t.setState(StHurt)
	return true, false
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

func sign(v float64) float64 {
	if v < 0 {
		return -1
	}
	return 1
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
