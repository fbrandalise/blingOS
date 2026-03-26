package main

import (
	"fmt"
	"hash/fnv"
	"strings"
)

// ── Pixel sprite engine ─────────────────────────────────────────
//
// Each sprite is a 2D grid of palette indices rendered with Unicode
// half-block characters (▀▄█). Two pixel rows fit in one terminal row
// using foreground/background colors, giving double vertical resolution.
//
// Palette:
//   0 = transparent
//   1 = outline (dark)
//   2 = skin tone
//   3 = accent (agent color)
//   4 = hair/hat
//   5 = prop/accessory
//   6 = white/highlight

type pixelSprite [][]int

const (
	pxClear     = 0
	pxLine      = 1
	pxSkin      = 2
	pxAccent    = 3
	pxHair      = 4
	pxProp      = 5
	pxHighlight = 6
)

// ── Unique character sprites ────────────────────────────────────
//
// Each character has a completely different silhouette, pose, and props.
// 14x14 grids — rendered to 14x7 terminal characters.
// Faces are deliberately squarish/blocky (Minecraft-ish).

// CEO: leaning back, sunglasses, confident stance, coffee in hand
var spriteCEO = pixelSprite{
	{0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0},
	{0, 0, 0, 1, 4, 4, 4, 4, 4, 4, 1, 0, 0, 0},
	{0, 0, 0, 1, 2, 2, 2, 2, 2, 2, 1, 0, 0, 0},
	{0, 0, 0, 1, 1, 1, 2, 2, 1, 1, 1, 0, 0, 0},
	{0, 0, 0, 1, 2, 2, 2, 2, 2, 2, 1, 0, 0, 0},
	{0, 0, 0, 0, 1, 2, 1, 1, 2, 1, 0, 0, 0, 0},
	{0, 0, 1, 3, 3, 3, 3, 3, 3, 3, 3, 1, 0, 0},
	{0, 1, 2, 3, 3, 3, 3, 3, 3, 3, 3, 2, 1, 0},
	{0, 0, 2, 2, 3, 3, 3, 3, 3, 3, 2, 5, 1, 0},
	{0, 0, 1, 2, 1, 3, 3, 3, 3, 1, 2, 5, 1, 0},
	{0, 0, 0, 1, 0, 1, 1, 1, 1, 0, 1, 5, 0, 0},
	{0, 0, 0, 1, 0, 0, 1, 1, 0, 0, 1, 0, 0, 0},
	{0, 0, 0, 1, 1, 0, 0, 0, 0, 1, 1, 0, 0, 0},
	{0, 0, 0, 1, 1, 0, 0, 0, 0, 1, 1, 0, 0, 0},
}

// PM: standing straight, clipboard in hand, organized look
var spritePM = pixelSprite{
	{0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0},
	{0, 0, 0, 1, 4, 4, 4, 4, 4, 4, 1, 0, 0, 0},
	{0, 0, 0, 1, 2, 2, 2, 2, 2, 2, 1, 0, 0, 0},
	{0, 0, 0, 1, 2, 1, 2, 2, 1, 2, 1, 0, 0, 0},
	{0, 0, 0, 1, 2, 2, 2, 2, 2, 2, 1, 0, 0, 0},
	{0, 0, 0, 0, 1, 2, 2, 2, 2, 1, 0, 0, 0, 0},
	{0, 0, 1, 3, 3, 3, 3, 3, 3, 3, 3, 1, 0, 0},
	{0, 1, 2, 3, 3, 3, 3, 3, 3, 3, 3, 2, 1, 0},
	{0, 0, 2, 3, 3, 3, 3, 3, 3, 3, 5, 5, 1, 0},
	{0, 0, 1, 2, 1, 3, 3, 3, 3, 1, 5, 5, 1, 0},
	{0, 0, 0, 1, 0, 0, 1, 1, 0, 0, 5, 5, 0, 0},
	{0, 0, 0, 1, 0, 0, 1, 1, 0, 0, 1, 0, 0, 0},
	{0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0},
	{0, 0, 0, 1, 1, 0, 0, 0, 0, 1, 1, 0, 0, 0},
}

// FE: hunched over laptop, typing furiously, hoodie
var spriteFE = pixelSprite{
	{0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0},
	{0, 0, 0, 1, 4, 4, 4, 4, 4, 4, 1, 0, 0, 0},
	{0, 0, 1, 4, 2, 2, 2, 2, 2, 2, 4, 1, 0, 0},
	{0, 0, 1, 4, 2, 1, 2, 2, 1, 2, 4, 1, 0, 0},
	{0, 0, 0, 1, 2, 2, 2, 2, 2, 2, 1, 0, 0, 0},
	{0, 0, 0, 0, 1, 2, 2, 2, 2, 1, 0, 0, 0, 0},
	{0, 0, 1, 3, 3, 3, 3, 3, 3, 3, 3, 1, 0, 0},
	{0, 1, 2, 3, 3, 3, 3, 3, 3, 3, 3, 2, 1, 0},
	{0, 2, 2, 5, 5, 5, 5, 5, 5, 5, 5, 2, 2, 0},
	{0, 0, 1, 5, 6, 6, 6, 6, 6, 6, 5, 1, 0, 0},
	{0, 0, 0, 5, 5, 5, 5, 5, 5, 5, 5, 0, 0, 0},
	{0, 0, 0, 1, 0, 0, 1, 1, 0, 0, 1, 0, 0, 0},
	{0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0},
	{0, 0, 0, 1, 1, 0, 0, 0, 0, 1, 1, 0, 0, 0},
}

// BE: arms crossed, slightly grumpy, server rack behind
var spriteBE = pixelSprite{
	{0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0},
	{0, 0, 0, 1, 4, 4, 4, 4, 4, 4, 1, 0, 0, 0},
	{0, 0, 0, 1, 2, 2, 2, 2, 2, 2, 1, 0, 0, 0},
	{0, 0, 0, 1, 2, 1, 2, 2, 1, 2, 1, 0, 0, 0},
	{0, 0, 0, 1, 2, 2, 1, 1, 2, 2, 1, 0, 0, 0},
	{0, 0, 0, 0, 1, 2, 2, 2, 2, 1, 0, 0, 0, 0},
	{0, 0, 1, 3, 3, 3, 3, 3, 3, 3, 3, 1, 0, 0},
	{0, 1, 2, 3, 3, 3, 3, 3, 3, 3, 3, 2, 1, 0},
	{0, 0, 1, 2, 3, 3, 3, 3, 3, 3, 2, 1, 0, 0},
	{0, 0, 1, 3, 2, 3, 3, 3, 3, 2, 3, 1, 0, 0},
	{0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 0},
	{0, 0, 0, 1, 0, 0, 1, 1, 0, 0, 1, 0, 0, 0},
	{0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0},
	{0, 0, 0, 1, 1, 0, 0, 0, 0, 1, 1, 0, 0, 0},
}

// AI: antenna on head, glowing eyes, slightly robotic
var spriteAI = pixelSprite{
	{0, 0, 0, 0, 0, 0, 5, 5, 0, 0, 0, 0, 0, 0},
	{0, 0, 0, 0, 0, 0, 1, 1, 0, 0, 0, 0, 0, 0},
	{0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0},
	{0, 0, 0, 1, 2, 2, 2, 2, 2, 2, 1, 0, 0, 0},
	{0, 0, 0, 1, 5, 1, 2, 2, 1, 5, 1, 0, 0, 0},
	{0, 0, 0, 1, 2, 2, 2, 2, 2, 2, 1, 0, 0, 0},
	{0, 0, 0, 0, 1, 2, 2, 2, 2, 1, 0, 0, 0, 0},
	{0, 0, 1, 3, 3, 3, 3, 3, 3, 3, 3, 1, 0, 0},
	{0, 1, 2, 3, 3, 3, 3, 3, 3, 3, 3, 2, 1, 0},
	{0, 0, 2, 2, 3, 3, 3, 3, 3, 3, 2, 2, 0, 0},
	{0, 0, 1, 2, 1, 3, 3, 3, 3, 1, 2, 1, 0, 0},
	{0, 0, 0, 1, 0, 0, 1, 1, 0, 0, 1, 0, 0, 0},
	{0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0},
	{0, 0, 0, 1, 1, 0, 0, 0, 0, 1, 1, 0, 0, 0},
}

// Designer: beret, holding pencil, creative pose
var spriteDesigner = pixelSprite{
	{0, 0, 0, 5, 5, 5, 5, 1, 0, 0, 0, 0, 0, 0},
	{0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0},
	{0, 0, 0, 1, 4, 4, 4, 4, 4, 4, 1, 0, 0, 0},
	{0, 0, 0, 1, 2, 2, 2, 2, 2, 2, 1, 0, 0, 0},
	{0, 0, 0, 1, 2, 1, 2, 2, 1, 2, 1, 0, 0, 0},
	{0, 0, 0, 1, 2, 2, 2, 2, 2, 2, 1, 0, 0, 0},
	{0, 0, 0, 0, 1, 2, 3, 3, 2, 1, 0, 0, 0, 0},
	{0, 0, 1, 3, 3, 3, 3, 3, 3, 3, 3, 1, 0, 0},
	{0, 1, 2, 3, 3, 3, 3, 3, 3, 3, 3, 2, 5, 0},
	{0, 0, 2, 2, 3, 3, 3, 3, 3, 3, 2, 2, 5, 0},
	{0, 0, 1, 2, 1, 3, 3, 3, 3, 1, 2, 1, 5, 0},
	{0, 0, 0, 1, 0, 0, 1, 1, 0, 0, 1, 0, 0, 0},
	{0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0},
	{0, 0, 0, 1, 1, 0, 0, 0, 0, 1, 1, 0, 0, 0},
}

// CMO: energetic pose, arms up, megaphone
var spriteCMO = pixelSprite{
	{0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0},
	{0, 0, 0, 1, 4, 4, 4, 4, 4, 4, 1, 0, 0, 0},
	{0, 0, 0, 1, 2, 2, 2, 2, 2, 2, 1, 0, 0, 0},
	{0, 0, 0, 1, 2, 1, 2, 2, 1, 2, 1, 0, 0, 0},
	{0, 0, 0, 1, 2, 2, 2, 2, 2, 2, 1, 0, 0, 0},
	{0, 0, 0, 0, 1, 2, 3, 3, 2, 1, 0, 0, 0, 0},
	{0, 0, 1, 3, 3, 3, 3, 3, 3, 3, 3, 1, 0, 0},
	{5, 1, 2, 3, 3, 3, 3, 3, 3, 3, 3, 2, 1, 0},
	{5, 5, 2, 2, 3, 3, 3, 3, 3, 3, 2, 2, 0, 0},
	{5, 0, 1, 2, 1, 3, 3, 3, 3, 1, 2, 1, 0, 0},
	{0, 0, 0, 1, 0, 0, 1, 1, 0, 0, 1, 0, 0, 0},
	{0, 0, 0, 1, 0, 0, 1, 1, 0, 0, 1, 0, 0, 0},
	{0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0},
	{0, 0, 0, 1, 1, 0, 0, 0, 0, 1, 1, 0, 0, 0},
}

// CRO: sharp look, briefcase, power stance
var spriteCRO = pixelSprite{
	{0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0},
	{0, 0, 0, 1, 4, 4, 4, 4, 4, 4, 1, 0, 0, 0},
	{0, 0, 0, 1, 2, 2, 2, 2, 2, 2, 1, 0, 0, 0},
	{0, 0, 0, 1, 2, 1, 2, 2, 1, 2, 1, 0, 0, 0},
	{0, 0, 0, 1, 2, 2, 2, 2, 2, 2, 1, 0, 0, 0},
	{0, 0, 0, 0, 1, 2, 2, 2, 2, 1, 0, 0, 0, 0},
	{0, 0, 1, 3, 6, 3, 3, 3, 3, 6, 3, 1, 0, 0},
	{0, 1, 2, 3, 6, 3, 3, 3, 3, 6, 3, 2, 1, 0},
	{0, 0, 2, 2, 3, 3, 3, 3, 3, 3, 2, 2, 0, 0},
	{0, 0, 1, 2, 1, 3, 3, 3, 3, 1, 2, 1, 0, 0},
	{0, 0, 0, 1, 0, 0, 1, 1, 0, 0, 1, 0, 0, 0},
	{0, 0, 0, 1, 0, 0, 1, 1, 0, 5, 5, 5, 0, 0},
	{0, 0, 0, 0, 1, 0, 0, 0, 0, 5, 1, 5, 0, 0},
	{0, 0, 0, 1, 1, 0, 0, 0, 0, 5, 5, 5, 0, 0},
}

// Generic agent — used for dynamically created agents
var spriteGeneric = pixelSprite{
	{0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0},
	{0, 0, 0, 1, 4, 4, 4, 4, 4, 4, 1, 0, 0, 0},
	{0, 0, 0, 1, 2, 2, 2, 2, 2, 2, 1, 0, 0, 0},
	{0, 0, 0, 1, 2, 1, 2, 2, 1, 2, 1, 0, 0, 0},
	{0, 0, 0, 1, 2, 2, 2, 2, 2, 2, 1, 0, 0, 0},
	{0, 0, 0, 0, 1, 2, 2, 2, 2, 1, 0, 0, 0, 0},
	{0, 0, 1, 3, 3, 3, 3, 3, 3, 3, 3, 1, 0, 0},
	{0, 1, 2, 3, 3, 3, 3, 3, 3, 3, 3, 2, 1, 0},
	{0, 0, 2, 2, 3, 3, 3, 3, 3, 3, 2, 2, 0, 0},
	{0, 0, 1, 2, 1, 3, 3, 3, 3, 1, 2, 1, 0, 0},
	{0, 0, 0, 1, 0, 0, 1, 1, 0, 0, 1, 0, 0, 0},
	{0, 0, 0, 1, 0, 0, 1, 1, 0, 0, 1, 0, 0, 0},
	{0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0},
	{0, 0, 0, 1, 1, 0, 0, 0, 0, 1, 1, 0, 0, 0},
}

// spriteForSlug returns the unique sprite for a known role,
// or a seeded variation of the generic sprite for dynamic agents.
func spriteForSlug(slug string) pixelSprite {
	switch slug {
	case "ceo":
		return spriteCEO
	case "pm":
		return spritePM
	case "fe":
		return spriteFE
	case "be":
		return spriteBE
	case "ai":
		return spriteAI
	case "designer":
		return spriteDesigner
	case "cmo":
		return spriteCMO
	case "cro":
		return spriteCRO
	default:
		// Dynamic agents get the generic sprite with seeded hair variation
		sprite := cloneSprite(spriteGeneric)
		applyHairVariation(sprite, seedHash(slug))
		return sprite
	}
}

func seedHash(s string) int {
	h := fnv.New32a()
	h.Write([]byte(s))
	return int(h.Sum32())
}

func applyHairVariation(sprite pixelSprite, seed int) {
	switch seed % 4 {
	case 0: // short crop
		sprite[0][6] = pxClear
		sprite[0][7] = pxClear
	case 1: // wider hair
		sprite[1][3] = pxHair
		sprite[1][10] = pxHair
	case 2: // tall hair
		if len(sprite) > 0 && len(sprite[0]) >= 10 {
			sprite[0][5] = pxHair
			sprite[0][8] = pxHair
		}
	default: // asymmetric
		sprite[0][5] = pxClear
		sprite[1][4] = pxHair
	}
}

func cloneSprite(src pixelSprite) pixelSprite {
	out := make(pixelSprite, len(src))
	for i := range src {
		out[i] = append([]int(nil), src[i]...)
	}
	return out
}

// ── Palette ─────────────────────────────────────────────────────

func parseHexColor(hex string) [3]int {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		return [3]int{140, 140, 150}
	}
	r, g, b := 0, 0, 0
	fmt.Sscanf(hex[0:2], "%x", &r)
	fmt.Sscanf(hex[2:4], "%x", &g)
	fmt.Sscanf(hex[4:6], "%x", &b)
	return [3]int{r, g, b}
}

func spritePaletteForSlug(slug string) map[int][3]int {
	accent := parseHexColor(agentColorMap[slug])
	if accent == ([3]int{}) {
		accent = [3]int{88, 166, 255}
	}
	// Hair color: darker version of accent
	hair := [3]int{
		max(0, accent[0]-60),
		max(0, accent[1]-60),
		max(0, accent[2]-60),
	}
	return map[int][3]int{
		pxLine:      {36, 32, 30},    // dark outline
		pxSkin:      {235, 215, 190}, // warm skin
		pxAccent:    accent,
		pxHair:      hair,
		pxProp:      {180, 170, 155}, // neutral prop color
		pxHighlight: {255, 255, 255}, // white highlights
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// ── Half-block renderer ─────────────────────────────────────────

func renderSpriteToANSI(sprite pixelSprite, palette map[int][3]int) []string {
	reset := "\x1b[0m"
	lines := make([]string, 0, (len(sprite)+1)/2)
	for r := 0; r < len(sprite); r += 2 {
		top := sprite[r]
		var bottom []int
		if r+1 < len(sprite) {
			bottom = sprite[r+1]
		}
		var b strings.Builder
		for c := 0; c < len(top); c++ {
			topVal := top[c]
			botVal := 0
			if bottom != nil && c < len(bottom) {
				botVal = bottom[c]
			}
			topRGB, topOK := palette[topVal]
			botRGB, botOK := palette[botVal]
			switch {
			case topVal != 0 && botVal != 0 && topOK && botOK:
				b.WriteString(fmt.Sprintf("\x1b[38;2;%d;%d;%dm\x1b[48;2;%d;%d;%dm\u2580%s",
					topRGB[0], topRGB[1], topRGB[2],
					botRGB[0], botRGB[1], botRGB[2], reset))
			case topVal != 0 && topOK:
				b.WriteString(fmt.Sprintf("\x1b[38;2;%d;%d;%dm\u2580%s",
					topRGB[0], topRGB[1], topRGB[2], reset))
			case botVal != 0 && botOK:
				b.WriteString(fmt.Sprintf("\x1b[38;2;%d;%d;%dm\u2584%s",
					botRGB[0], botRGB[1], botRGB[2], reset))
			default:
				b.WriteByte(' ')
			}
		}
		lines = append(lines, b.String())
	}
	return lines
}

// ── Public API ──────────────────────────────────────────────────

// renderWuphfSplashAvatar renders a full-body character for the splash screen.
func renderWuphfSplashAvatar(seed, slug string, talking bool) []string {
	_ = seed
	_ = talking
	sprite := spriteForSlug(slug)
	return renderSpriteToANSI(sprite, spritePaletteForSlug(slug))
}

// renderWuphfAvatar renders a small face portrait for inline use.
func renderWuphfAvatar(seed, slug string, talking bool) []string {
	_ = seed
	_ = talking
	// Use just the head portion (rows 0-5) of the full sprite
	full := spriteForSlug(slug)
	if len(full) > 6 {
		head := full[:6]
		return renderSpriteToANSI(head, spritePaletteForSlug(slug))
	}
	return renderSpriteToANSI(full, spritePaletteForSlug(slug))
}
