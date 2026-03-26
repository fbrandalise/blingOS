package main

import (
	"fmt"
	"hash/fnv"
	"strconv"
	"strings"
)

type pixelSprite [][]int

type pixelUpdate struct {
	row int
	col int
	val int
}

const (
	pxClear  = 0
	pxLine   = 1
	pxSkin   = 2
	pxAccent = 3
)

var humanIdleSprite = pixelSprite{
	{0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 0, 0, 0, 0, 0, 0},
	{0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0, 0},
	{0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0},
	{0, 0, 0, 1, 1, 2, 2, 2, 2, 2, 2, 1, 1, 0, 0, 0},
	{0, 0, 0, 1, 2, 2, 1, 2, 2, 1, 2, 2, 1, 0, 0, 0},
	{0, 0, 0, 1, 2, 2, 2, 2, 2, 2, 2, 2, 1, 0, 0, 0},
	{0, 0, 0, 1, 1, 2, 2, 1, 1, 2, 2, 1, 1, 0, 0, 0},
	{0, 0, 1, 3, 3, 1, 2, 2, 2, 2, 1, 3, 3, 1, 0, 0},
	{0, 1, 3, 3, 3, 3, 1, 3, 3, 1, 3, 3, 3, 3, 1, 0},
	{0, 1, 2, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 2, 1, 0},
	{0, 1, 2, 2, 3, 3, 3, 3, 3, 3, 3, 3, 2, 2, 1, 0},
	{0, 0, 1, 2, 2, 1, 3, 3, 3, 3, 1, 2, 2, 1, 0, 0},
	{0, 0, 1, 2, 2, 1, 0, 3, 3, 0, 1, 2, 2, 1, 0, 0},
	{0, 0, 1, 1, 2, 1, 0, 1, 1, 0, 1, 2, 1, 1, 0, 0},
	{0, 0, 0, 1, 2, 1, 0, 1, 1, 0, 1, 2, 1, 0, 0, 0},
	{0, 0, 0, 1, 1, 0, 0, 1, 1, 0, 0, 1, 1, 0, 0, 0},
}

var humanTalkSprite = pixelSprite{
	{0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 0, 0, 0, 0, 0, 0},
	{0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0, 0},
	{0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0},
	{0, 0, 0, 1, 1, 2, 2, 2, 2, 2, 2, 1, 1, 0, 0, 0},
	{0, 0, 0, 1, 2, 2, 1, 2, 2, 1, 2, 2, 1, 0, 0, 0},
	{0, 0, 0, 1, 2, 2, 2, 2, 2, 2, 2, 2, 1, 0, 0, 0},
	{0, 0, 0, 1, 1, 2, 2, 1, 2, 1, 2, 1, 1, 0, 0, 0},
	{0, 0, 1, 3, 3, 1, 2, 2, 2, 2, 1, 3, 3, 1, 0, 0},
	{0, 1, 3, 3, 3, 3, 1, 3, 3, 1, 3, 3, 3, 3, 1, 0},
	{0, 1, 2, 3, 3, 3, 3, 1, 1, 3, 3, 3, 3, 2, 1, 0},
	{0, 1, 2, 2, 3, 3, 3, 2, 2, 3, 3, 3, 2, 2, 1, 0},
	{0, 0, 1, 2, 2, 1, 3, 3, 3, 3, 1, 2, 2, 1, 1, 0},
	{0, 0, 1, 2, 2, 1, 0, 3, 3, 0, 1, 2, 0, 1, 2, 1},
	{0, 0, 1, 1, 2, 1, 0, 1, 1, 0, 1, 2, 0, 0, 1, 0},
	{0, 0, 0, 1, 2, 1, 0, 1, 1, 0, 1, 2, 1, 0, 0, 0},
	{0, 0, 0, 1, 1, 0, 0, 1, 1, 0, 0, 1, 1, 0, 0, 0},
}

var humanFaceSprite = pixelSprite{
	{0, 0, 0, 1, 1, 1, 1, 1, 1, 0, 0, 0},
	{0, 0, 1, 1, 2, 2, 2, 2, 1, 1, 0, 0},
	{0, 1, 2, 2, 1, 2, 2, 1, 2, 2, 1, 0},
	{0, 1, 2, 2, 2, 2, 2, 2, 2, 2, 1, 0},
	{0, 0, 1, 3, 3, 3, 3, 3, 3, 1, 0, 0},
	{0, 1, 3, 3, 1, 3, 3, 1, 3, 3, 1, 0},
}

func parseHexColor(hex string) [3]int {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		return [3]int{140, 140, 150}
	}
	r, _ := strconv.ParseInt(hex[0:2], 16, 0)
	g, _ := strconv.ParseInt(hex[2:4], 16, 0)
	b, _ := strconv.ParseInt(hex[4:6], 16, 0)
	return [3]int{int(r), int(g), int(b)}
}

func lightenRGB(rgb [3]int, delta int) [3]int {
	return [3]int{
		min(255, rgb[0]+delta),
		min(255, rgb[1]+delta),
		min(255, rgb[2]+delta),
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func cloneSprite(src pixelSprite) pixelSprite {
	out := make(pixelSprite, len(src))
	for i := range src {
		out[i] = append([]int(nil), src[i]...)
	}
	return out
}

func applyPixels(dst pixelSprite, updates ...pixelUpdate) {
	for _, u := range updates {
		if u.row < 0 || u.row >= len(dst) || u.col < 0 || u.col >= len(dst[u.row]) {
			continue
		}
		dst[u.row][u.col] = u.val
	}
}

func seedTrait(seed string) int {
	h := fnv.New32a()
	_, _ = h.Write([]byte(seed))
	return int(h.Sum32())
}

func applyHair(sprite pixelSprite, style int) {
	switch style % 4 {
	case 0:
		applyPixels(sprite,
			pixelUpdate{0, 6, pxClear}, pixelUpdate{0, 9, pxClear},
			pixelUpdate{1, 5, pxClear},
		)
	case 1:
		applyPixels(sprite,
			pixelUpdate{0, 5, pxLine}, pixelUpdate{0, 10, pxLine},
			pixelUpdate{2, 4, pxLine}, pixelUpdate{2, 11, pxLine},
		)
	case 2:
		applyPixels(sprite,
			pixelUpdate{1, 4, pxLine}, pixelUpdate{1, 5, pxLine}, pixelUpdate{1, 10, pxLine}, pixelUpdate{1, 11, pxLine},
			pixelUpdate{2, 3, pxLine}, pixelUpdate{2, 12, pxLine},
		)
	default:
		applyPixels(sprite,
			pixelUpdate{0, 7, pxClear}, pixelUpdate{0, 8, pxClear},
			pixelUpdate{1, 6, pxClear}, pixelUpdate{1, 9, pxClear},
		)
	}
}

func applyRole(sprite pixelSprite, slug string) {
	switch slug {
	case "ceo":
		applyPixels(sprite,
			pixelUpdate{3, 9, pxLine},
			pixelUpdate{7, 7, pxSkin}, pixelUpdate{7, 8, pxSkin},
			pixelUpdate{9, 7, pxAccent}, pixelUpdate{10, 7, pxSkin}, pixelUpdate{11, 7, pxAccent},
		)
	case "pm":
		applyPixels(sprite,
			pixelUpdate{8, 12, pxSkin}, pixelUpdate{8, 13, pxSkin},
			pixelUpdate{9, 11, pxSkin}, pixelUpdate{9, 12, pxSkin}, pixelUpdate{9, 13, pxSkin},
			pixelUpdate{10, 11, pxLine}, pixelUpdate{10, 12, pxAccent}, pixelUpdate{10, 13, pxAccent},
			pixelUpdate{11, 11, pxLine}, pixelUpdate{11, 12, pxAccent}, pixelUpdate{11, 13, pxAccent},
		)
	case "fe":
		applyPixels(sprite,
			pixelUpdate{6, 3, pxAccent}, pixelUpdate{6, 12, pxAccent},
			pixelUpdate{7, 2, pxAccent}, pixelUpdate{7, 13, pxAccent},
			pixelUpdate{8, 2, pxAccent}, pixelUpdate{8, 13, pxAccent},
		)
	case "be":
		applyPixels(sprite,
			pixelUpdate{6, 4, pxLine}, pixelUpdate{6, 11, pxLine},
			pixelUpdate{7, 5, pxLine}, pixelUpdate{7, 10, pxLine},
			pixelUpdate{11, 6, pxAccent}, pixelUpdate{11, 9, pxAccent},
		)
	case "ai":
		applyPixels(sprite,
			pixelUpdate{0, 7, pxAccent},
			pixelUpdate{1, 7, pxAccent},
			pixelUpdate{2, 7, pxAccent},
			pixelUpdate{1, 8, pxAccent},
		)
	case "designer":
		applyPixels(sprite,
			pixelUpdate{4, 4, pxLine}, pixelUpdate{4, 5, pxLine}, pixelUpdate{4, 9, pxLine}, pixelUpdate{4, 10, pxLine},
			pixelUpdate{8, 6, pxAccent}, pixelUpdate{8, 7, pxAccent}, pixelUpdate{8, 8, pxAccent}, pixelUpdate{8, 9, pxAccent},
		)
	case "cmo":
		applyPixels(sprite,
			pixelUpdate{4, 4, pxLine}, pixelUpdate{4, 5, pxLine}, pixelUpdate{4, 9, pxLine}, pixelUpdate{4, 10, pxLine},
			pixelUpdate{7, 3, pxAccent}, pixelUpdate{7, 12, pxAccent},
			pixelUpdate{8, 3, pxAccent}, pixelUpdate{8, 12, pxAccent},
		)
	case "cro":
		applyPixels(sprite,
			pixelUpdate{10, 11, pxAccent}, pixelUpdate{10, 12, pxAccent},
			pixelUpdate{11, 11, pxAccent}, pixelUpdate{11, 12, pxAccent},
			pixelUpdate{12, 11, pxLine}, pixelUpdate{12, 12, pxLine},
		)
	}
}

func applyFaceHair(sprite pixelSprite, style int) {
	switch style % 5 {
	case 0:
		applyPixels(sprite,
			pixelUpdate{0, 3, pxClear}, pixelUpdate{0, 8, pxClear},
			pixelUpdate{1, 2, pxClear}, pixelUpdate{1, 9, pxClear},
		)
	case 1:
		applyPixels(sprite,
			pixelUpdate{0, 2, pxLine}, pixelUpdate{0, 9, pxLine},
			pixelUpdate{1, 1, pxLine}, pixelUpdate{1, 10, pxLine},
		)
	case 2:
		applyPixels(sprite,
			pixelUpdate{0, 4, pxClear}, pixelUpdate{0, 5, pxClear},
			pixelUpdate{1, 4, pxClear},
		)
	case 3:
		applyPixels(sprite,
			pixelUpdate{0, 6, pxClear}, pixelUpdate{0, 7, pxClear},
			pixelUpdate{1, 7, pxClear},
		)
	default:
		applyPixels(sprite,
			pixelUpdate{0, 2, pxLine}, pixelUpdate{0, 3, pxLine}, pixelUpdate{0, 8, pxLine}, pixelUpdate{0, 9, pxLine},
		)
	}
}

func applyRoleFace(sprite pixelSprite, slug string) {
	switch slug {
	case "ceo":
		applyPixels(sprite,
			pixelUpdate{0, 5, pxClear}, pixelUpdate{0, 6, pxClear},
			pixelUpdate{1, 5, pxSkin}, pixelUpdate{1, 6, pxSkin},
			pixelUpdate{4, 5, pxSkin}, pixelUpdate{4, 6, pxSkin},
			pixelUpdate{5, 5, pxAccent}, pixelUpdate{5, 6, pxAccent},
		)
	case "pm":
		applyPixels(sprite,
			pixelUpdate{1, 1, pxAccent}, pixelUpdate{2, 1, pxAccent}, pixelUpdate{3, 1, pxAccent},
			pixelUpdate{4, 8, pxSkin}, pixelUpdate{4, 9, pxSkin},
		)
	case "fe":
		applyPixels(sprite,
			pixelUpdate{1, 2, pxAccent}, pixelUpdate{1, 9, pxAccent},
			pixelUpdate{4, 2, pxAccent}, pixelUpdate{4, 9, pxAccent},
			pixelUpdate{5, 1, pxAccent}, pixelUpdate{5, 10, pxAccent},
		)
	case "be":
		applyPixels(sprite,
			pixelUpdate{2, 4, pxLine}, pixelUpdate{2, 7, pxLine},
			pixelUpdate{3, 4, pxLine}, pixelUpdate{3, 7, pxLine},
			pixelUpdate{5, 4, pxAccent}, pixelUpdate{5, 7, pxAccent},
		)
	case "ai":
		applyPixels(sprite,
			pixelUpdate{0, 5, pxAccent},
			pixelUpdate{1, 5, pxAccent},
			pixelUpdate{1, 6, pxAccent},
		)
	case "designer":
		applyPixels(sprite,
			pixelUpdate{2, 3, pxLine}, pixelUpdate{2, 4, pxLine}, pixelUpdate{2, 7, pxLine}, pixelUpdate{2, 8, pxLine},
			pixelUpdate{4, 3, pxAccent}, pixelUpdate{4, 8, pxAccent},
			pixelUpdate{5, 2, pxAccent}, pixelUpdate{5, 9, pxAccent},
		)
	case "cmo":
		applyPixels(sprite,
			pixelUpdate{2, 3, pxLine}, pixelUpdate{2, 4, pxLine}, pixelUpdate{2, 7, pxLine}, pixelUpdate{2, 8, pxLine},
			pixelUpdate{0, 2, pxAccent}, pixelUpdate{0, 9, pxAccent},
			pixelUpdate{1, 2, pxAccent}, pixelUpdate{1, 9, pxAccent},
		)
	case "cro":
		applyPixels(sprite,
			pixelUpdate{4, 7, pxSkin}, pixelUpdate{4, 8, pxSkin},
			pixelUpdate{5, 7, pxAccent}, pixelUpdate{5, 8, pxAccent},
		)
	}
}

func humanSprite(seed, slug string, talking bool) pixelSprite {
	var src pixelSprite
	if talking {
		src = humanTalkSprite
	} else {
		src = humanIdleSprite
	}
	sprite := cloneSprite(src)
	applyHair(sprite, seedTrait(seed+"|hair"))
	applyRole(sprite, slug)
	return sprite
}

func humanFacePortrait(seed, slug string) pixelSprite {
	sprite := cloneSprite(humanFaceSprite)
	applyFaceHair(sprite, seedTrait(seed+"|portrait"))
	applyRoleFace(sprite, slug)
	return sprite
}

func scaleSprite(src pixelSprite, outW, outH int) pixelSprite {
	if len(src) == 0 || len(src[0]) == 0 || outW <= 0 || outH <= 0 {
		return nil
	}
	inH, inW := len(src), len(src[0])
	out := make(pixelSprite, outH)
	for y := 0; y < outH; y++ {
		out[y] = make([]int, outW)
		srcY := y * inH / outH
		for x := 0; x < outW; x++ {
			srcX := x * inW / outW
			out[y][x] = src[srcY][srcX]
		}
	}
	return out
}

func spritePaletteForSlug(slug string) map[int][3]int {
	accent := parseHexColor(agentColorMap[slug])
	if accent == [3]int{} {
		accent = [3]int{88, 166, 255}
	}
	return map[int][3]int{
		pxLine:   [3]int{46, 40, 39},
		pxSkin:   [3]int{241, 223, 201},
		pxAccent: accent,
	}
}

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
			if bottom != nil {
				botVal = bottom[c]
			}
			topRGB, topOK := palette[topVal]
			botRGB, botOK := palette[botVal]
			switch {
			case topVal != 0 && botVal != 0 && topOK && botOK:
				b.WriteString(fmt.Sprintf("\x1b[38;2;%d;%d;%dm\x1b[48;2;%d;%d;%dm▀%s", topRGB[0], topRGB[1], topRGB[2], botRGB[0], botRGB[1], botRGB[2], reset))
			case topVal != 0 && topOK:
				b.WriteString(fmt.Sprintf("\x1b[38;2;%d;%d;%dm▀%s", topRGB[0], topRGB[1], topRGB[2], reset))
			case botVal != 0 && botOK:
				b.WriteString(fmt.Sprintf("\x1b[38;2;%d;%d;%dm▄%s", botRGB[0], botRGB[1], botRGB[2], reset))
			default:
				b.WriteByte(' ')
			}
		}
		lines = append(lines, b.String())
	}
	return lines
}

func renderWuphfAvatar(seed, slug string, talking bool) []string {
	_ = talking
	sprite := humanFacePortrait(seed, slug)
	return renderSpriteToANSI(scaleSprite(sprite, 4, 4), spritePaletteForSlug(slug))
}

func renderWuphfSplashAvatar(seed, slug string, talking bool) []string {
	sprite := humanSprite(seed, slug, talking)
	return renderSpriteToANSI(sprite, spritePaletteForSlug(slug))
}
