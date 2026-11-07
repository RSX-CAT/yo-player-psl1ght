package renpy

import (
	"regexp"
	"strconv"
	"strings"
)

type ImageMapHotspot struct {
	X0, Y0, X1, Y1 int
	Name           string
}

type ImageMapDef struct {
	Kind          string
	Ground        string
	Idle          string
	Hover         string
	SelectedIdle  string
	SelectedHover string
	Hotspots      []ImageMapHotspot
}

func (d ImageMapDef) Key() string {
	var b strings.Builder
	b.WriteString(d.Kind)
	b.WriteByte('|')
	b.WriteString(d.Ground)
	b.WriteByte('|')
	b.WriteString(d.Idle)
	b.WriteByte('|')
	b.WriteString(d.Hover)
	b.WriteByte('|')
	b.WriteString(d.SelectedIdle)
	b.WriteByte('|')
	b.WriteString(d.SelectedHover)
	b.WriteByte('|')
	for _, h := range d.Hotspots {
		b.WriteString(strconv.Itoa(h.X0))
		b.WriteByte(',')
		b.WriteString(strconv.Itoa(h.Y0))
		b.WriteByte(',')
		b.WriteString(strconv.Itoa(h.X1))
		b.WriteByte(',')
		b.WriteString(strconv.Itoa(h.Y1))
		b.WriteByte('=')
		b.WriteString(h.Name)
		b.WriteByte(';')
	}
	return b.String()
}

var (
	imGroundHover = regexp.MustCompile(`(?s)renpy\.imagemap\(\s*[uU]?["']([^"']+)["']\s*,\s*[uU]?["']([^"']+)["']`)
	imThemed      = regexp.MustCompile(`(?s)layout\.imagemap_(main_menu|navigation|preferences|yesno_prompt|load_save)\s*\(`)
	imHotspot     = regexp.MustCompile(`(?s)\(\s*(\d+)\s*,\s*(\d+)\s*,\s*(\d+)\s*,\s*(\d+)\s*,\s*[uU]?["']([^"']+)["']\s*\)`)
	imStrArg      = regexp.MustCompile(`(?s)[uU]?["']([^"']*)["']`)
)

func ParseImageMap(src string) *ImageMapDef {
	if src == "" {
		return nil
	}
	gh := imGroundHover.FindStringSubmatchIndex(src)
	if gh == nil {
		return nil
	}
	d := &ImageMapDef{
		Ground: src[gh[2]:gh[3]],
		Hover:  src[gh[4]:gh[5]],
	}
	addHotspots(d, src[gh[0]:])
	if len(d.Hotspots) == 0 {
		return nil
	}
	return d
}

func ParseThemedImageMaps(src string) []ImageMapDef {
	var result []ImageMapDef
	if src == "" {
		return result
	}
	matches := imThemed.FindAllStringSubmatchIndex(src, -1)
	for _, m := range matches {
		screen := src[m[2]:m[3]]
		open := m[1] - 1
		span := matchParens(src, open)
		if span < 0 {
			continue
		}
		args := src[open+1 : span]
		br := strings.IndexByte(args, '[')
		head := args
		if br >= 0 {
			head = args[:br]
		}
		var imgs []string
		for _, s := range imStrArg.FindAllStringSubmatch(head, -1) {
			imgs = append(imgs, s[1])
		}
		d := ImageMapDef{Kind: screen}
		assignImages(&d, screen, imgs)
		if br >= 0 {
			addHotspots(&d, args[br:])
		}
		if len(d.Hotspots) > 0 {
			result = append(result, d)
		}
	}
	return result
}

func assignImages(d *ImageMapDef, screen string, imgs []string) {
	ground := ""
	if len(imgs) > 0 {
		ground = imgs[0]
	}
	d.Ground = ground
	switch screen {
	case "main_menu":
		d.Idle = ground
		d.Hover = ground
		if len(imgs) > 1 {
			d.SelectedIdle = imgs[1]
		} else {
			d.SelectedIdle = ground
		}
		d.SelectedHover = d.SelectedIdle
	case "yesno_prompt":
		if len(imgs) > 1 {
			d.Idle = imgs[1]
		} else {
			d.Idle = ground
		}
		if len(imgs) > 2 {
			d.Hover = imgs[2]
		} else {
			d.Hover = d.Idle
		}
		d.SelectedIdle = d.Idle
		d.SelectedHover = d.Hover
	default:
		if len(imgs) > 1 {
			d.Idle = imgs[1]
		} else {
			d.Idle = ground
		}
		if len(imgs) > 2 {
			d.Hover = imgs[2]
		} else {
			d.Hover = d.Idle
		}
		if len(imgs) > 3 {
			d.SelectedIdle = imgs[3]
		} else {
			d.SelectedIdle = d.Idle
		}
		if len(imgs) > 4 {
			d.SelectedHover = imgs[4]
		} else {
			d.SelectedHover = d.Hover
		}
	}
}

func addHotspots(d *ImageMapDef, src string) {
	for _, m := range imHotspot.FindAllStringSubmatch(src, -1) {
		x0, _ := strconv.Atoi(m[1])
		y0, _ := strconv.Atoi(m[2])
		x1, _ := strconv.Atoi(m[3])
		y1, _ := strconv.Atoi(m[4])
		d.Hotspots = append(d.Hotspots, ImageMapHotspot{X0: x0, Y0: y0, X1: x1, Y1: y1, Name: m[5]})
	}
}

func matchParens(s string, openIndex int) int {
	depth := 0
	inStr := false
	var q byte
	for i := openIndex; i < len(s); i++ {
		c := s[i]
		if inStr {
			if c == q {
				inStr = false
			}
			continue
		}
		if c == '"' || c == '\'' {
			inStr = true
			q = c
			continue
		}
		if c == '(' {
			depth++
		} else if c == ')' {
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}
