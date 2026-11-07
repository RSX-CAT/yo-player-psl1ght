package renpy

import (
	"fmt"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

func BuildGuiManifest(p *IrProgram) string {
	var lines []string
	for _, s := range p.Strings {
		if strings.Contains(s, "style.") || strings.Contains(s, "gui.") ||
			strings.Contains(s, "config.") || strings.Contains(s, "theme.") ||
			strings.Contains(s, "Character(") {
			for _, ln := range strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n") {
				lines = append(lines, strings.TrimSpace(ln))
			}
		}
	}
	f := map[string]string{}

	for _, ln := range lines {
		if tm := reWindowTitle.FindStringSubmatch(ln); tm != nil {
			f["title"] = strings.TrimSpace(tm[1])
		}
		if mm := reMainMenuMusic.FindStringSubmatch(ln); mm != nil {
			f["mm_music"] = strings.TrimSpace(mm[1])
		}
	}

	modern := hasGui(lines)
	if !modern {
		f["textbox_xmargin"] = "10"
		f["textbox_margin_t"] = "5"
		f["textbox_margin_b"] = "5"
		f["textbox_pad_x"] = "10"
		f["textbox_pad_y"] = "5"
		f["textbox_ymin"] = "150"
		f["textbox_color"] = "#00008080"
		f["name_spacing"] = "8"
		f["name_bold"] = "1"
		f["nvl_pad_x"] = "20"
		f["nvl_pad_y"] = "30"
		f["nvl_spacing"] = "10"
		f["nvl_bg"] = "#0008"
		usesTheme, rounded, roundedSeen := false, true, false
		for _, ln := range lines {
			if reThemeCall.MatchString(ln) {
				usesTheme = true
			}
			if rm := reRoundedWindow.FindStringSubmatch(ln); rm != nil {
				rounded = rm[1] == "True"
				roundedSeen = true
			}
		}
		if usesTheme {
			f["textbox_pad_x"] = "6"
			f["textbox_pad_y"] = "6"
			m := "6"
			if roundedSeen && !rounded {
				m = "0"
			}
			f["textbox_xmargin"] = m
			f["textbox_margin_t"] = m
			f["textbox_margin_b"] = m
			delete(f, "textbox_color")
			p.Notes = append(p.Notes, "theme window background is engine-generated (not extractable); textbox shows the game's own background override if present")
		}
	}

	for _, ln := range lines {
		grab(ln, reScreenW, f, "native_w", fmtRaw)
		grab(ln, reScreenH, f, "native_h", fmtRaw)
		grab(ln, reThumbW, f, "thumb_w", fmtRaw)
		grab(ln, reThumbH, f, "thumb_h", fmtRaw)
		grab(ln, reTextCps, f, "text_cps", fmtRaw)
		matchTwo(ln, reGuiInit, f, "native_w", "native_h")
		matchScriptVersion(ln, f)

		grab(ln, reStyleFont, f, "text_font", fmtName)
		grab(ln, reStyleSize, f, "text_size", fmtRaw)
		grab(ln, reStyleColor, f, "text_color", fmtColor)
		matchTwoJoin(ln, reDropShadow, f, "text_shadow")
		grab(ln, reDropShadowColor, f, "text_shadow_color", fmtColor)
		grab(ln, reSayLabelSize, f, "name_size", fmtRaw)
		grab(ln, reSayLabelColor, f, "name_color", fmtColor)

		matchFrame(ln, f)
		matchWindowColor(ln, f)
		grab(ln, reWindowBgImg, f, "textbox_bg", fmtName)
		grab(ln, reBottomMargin, f, "textbox_margin_b", fmtRaw)
		grab(ln, reTopMargin, f, "textbox_margin_t", fmtRaw)
		matchYmargin(ln, f)
		grab(ln, reXpadding, f, "textbox_pad_x", fmtRaw)
		grab(ln, reYpadding, f, "textbox_pad_y", fmtRaw)
		grab(ln, reLeftPad, f, "textbox_pad_l", fmtRaw)
		grab(ln, reRightPad, f, "textbox_pad_r", fmtRaw)
		grab(ln, reTopPad, f, "textbox_pad_t", fmtRaw)
		grab(ln, reBottomPad, f, "textbox_pad_b", fmtRaw)
		grab(ln, reXmargin, f, "textbox_xmargin", fmtRaw)
		grab(ln, reYmin, f, "textbox_ymin", fmtRaw)

		grab(ln, reCtcAnim, f, "ctc", fmtName)
		grab(ln, reGuiCtc, f, "ctc", fmtName)
		grab(ln, reCtcXpos, f, "ctc_xpos", fmtRaw)
		grab(ln, reCtcYpos, f, "ctc_ypos", fmtRaw)
		grab(ln, reCtcXanchor, f, "ctc_xanchor", fmtRaw)
		grab(ln, reCtcYanchor, f, "ctc_yanchor", fmtRaw)
		if reCtcFixed.MatchString(ln) {
			f["ctc_fixed"] = "1"
		}

		grab(ln, reMmRoot, f, "mm_bg", fmtName)
		grab(ln, reGmRoot, f, "gm_bg", fmtName)
		matchThemeColor(ln, "frame", f, "mm_frame_color")
		matchThemeColor(ln, "widget", f, "gm_btn_idle")
		matchThemeColor(ln, "widget_hover", f, "gm_btn_hover")
		matchThemeColor(ln, "widget_text", f, "gm_btn_text")
		matchThemeColor(ln, "widget_selected", f, "gm_btn_selected")
		matchThemeColor(ln, "disabled", f, "gm_btn_disabled")
		matchThemeColor(ln, "disabled_text", f, "gm_btn_disabled_text")
		grab(ln, reTextSizeKw, f, "gm_text_size", fmtRaw)
		if reMmFrameNone.MatchString(ln) {
			f["mm_frame_none"] = "1"
		}
		if ib := reImageButton.FindStringSubmatch(ln); ib != nil {
			f["mm_btn."+ib[1]] = ib[2] + "|" + ib[3] + "|" + ib[4] + "|" + ib[6]
		}

		grab(ln, reNvlBg, f, "nvl_bg", fmtColorA)
		grab(ln, reNvlXpad, f, "nvl_pad_x", fmtRaw)
		grab(ln, reNvlYpad, f, "nvl_pad_y", fmtRaw)
		grab(ln, reNvlSpacing, f, "nvl_spacing", fmtRaw)
		grab(ln, reSayVboxSpacing, f, "name_spacing", fmtRaw)
		if reSayLabelBoldFalse.MatchString(ln) {
			f["name_bold"] = "0"
		} else if reSayLabelBoldTrue.MatchString(ln) {
			f["name_bold"] = "1"
		}
		grab(ln, reSlotTextSize, f, "slot_text_size", fmtRaw)
		grab(ln, reSlotTextX, f, "slot_text_x", fmtRaw)
		grab(ln, reSlotTextY, f, "slot_text_y", fmtRaw)
		grab(ln, reSlotSsX, f, "slot_ss_x", fmtRaw)
		grab(ln, reSlotSsY, f, "slot_ss_y", fmtRaw)
		grab(ln, reSlotTextColor, f, "slot_text_color", fmtColorA)

		matchChoiceFrame(ln, "background", f, "choice_bg", "choice_frame")
		matchChoiceFrame(ln, "hover_background", f, "choice_hover_bg", "choice_frame")
		grab(ln, reChoiceSize, f, "choice_size", fmtRaw)
		grab(ln, reChoiceXmin, f, "choice_xmin", fmtRaw)
		grab(ln, reChoiceYmin, f, "choice_ymin", fmtRaw)
		grab(ln, reChoiceMargin, f, "choice_margin", fmtRaw)
		grab(ln, reMenuYalign, f, "menu_yalign", fmtRaw)

		grab(ln, reGuiTextSize, f, "text_size", fmtRaw)
		grab(ln, reGuiTextFont, f, "text_font", fmtName)
		grab(ln, reGuiTextColor, f, "text_color", fmtColor)
		grab(ln, reGuiNameSize, f, "name_size", fmtRaw)
		grab(ln, reGuiTextboxH, f, "textbox_height", fmtRaw)
		matchCharColor(ln, f)
	}

	extractIngameBox(p, f)
	extractMenuOrder(lines, f)
	extractSideImages(p, f)
	extractTwoWindow(lines, p, f)

	if _, ok := f["choice_color"]; !ok {
		if tc, ok := f["text_color"]; ok {
			f["choice_color"] = tc
		}
	}
	if _, ok := f["choice_hover_color"]; !ok {
		if cc, ok := f["choice_color"]; ok {
			f["choice_hover_color"] = cc
		}
	}
	if _, ok := f["textbox_bg"]; !ok {
		if _, ok := f["textbox_height"]; ok || hasGui(lines) {
			f["textbox_bg"] = "textbox.png"
		}
	}

	keys := make([]string, 0, len(f))
	for k := range f {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var sb strings.Builder
	for _, k := range keys {
		if !strings.HasPrefix(k, "char_color.") {
			sb.WriteString(k)
			sb.WriteByte('=')
			sb.WriteString(f[k])
			sb.WriteByte('\n')
		}
	}
	for _, k := range keys {
		if strings.HasPrefix(k, "char_color.") {
			sb.WriteString(k)
			sb.WriteByte('=')
			sb.WriteString(f[k])
			sb.WriteByte('\n')
		}
	}
	return sb.String()
}

func hasGui(lines []string) bool {
	for _, l := range lines {
		if strings.HasPrefix(l, "gui.") {
			return true
		}
	}
	return false
}

func extractMenuOrder(lines []string, f map[string]string) {
	main := []string{"Start Game", "Load Game", "Preferences", "Help", "Quit"}
	game := []string{"Return", "Preferences", "Save Game", "Load Game", "Main Menu", "Help", "Quit"}
	helpOn := false
	for _, ln := range lines {
		if reHelpOn.MatchString(ln) {
			helpOn = true
		}
		applyMenuMutation(ln, "main_menu", &main)
		applyMenuMutation(ln, "game_menu", &game)
	}
	if !helpOn {
		main = removeStr(main, "Help")
		game = removeStr(game, "Help")
	}
	f["mm_order"] = strings.Join(main, "|")
	f["gm_order"] = strings.Join(game, "|")
}

func applyMenuMutation(ln, which string, list *[]string) {
	reIns := regexp.MustCompile(`config\.` + which + `\.insert\(\s*(\d+)\s*,\s*\(\s*u?["']([^"']+)["']`)
	if m := reIns.FindStringSubmatch(ln); m != nil {
		idx, _ := strconv.Atoi(m[1])
		if idx < 0 {
			idx = 0
		}
		if idx > len(*list) {
			idx = len(*list)
		}
		if !containsStr(*list, m[2]) {
			*list = append((*list)[:idx], append([]string{m[2]}, (*list)[idx:]...)...)
		}
		return
	}
	reApp := regexp.MustCompile(`config\.` + which + `\.append\(\s*\(\s*u?["']([^"']+)["']`)
	if m := reApp.FindStringSubmatch(ln); m != nil {
		if !containsStr(*list, m[1]) {
			*list = append(*list, m[1])
		}
		return
	}
	reRem := regexp.MustCompile(`config\.` + which + `\.remove\([^)]*["']([^"']+)["']`)
	if m := reRem.FindStringSubmatch(ln); m != nil {
		*list = removeStr(*list, m[1])
	}
}

func extractIngameBox(p *IrProgram, f map[string]string) {
	for _, s := range p.Strings {
		idx := 0
		for {
			i := strings.Index(s[idx:], "Character(")
			if i < 0 {
				break
			}
			i += idx
			open := i + len("Character")
			call := balancedCall(s, open)
			idx = i + 1
			if !strings.Contains(call, "window_background") {
				continue
			}
			bg := reWinBg.FindStringSubmatch(call)
			if bg == nil {
				continue
			}
			f["ig_textbox_bg"] = baseName(bg[1])
			grabInt(call, reWinLPad, f, "ig_pad_l")
			grabInt(call, reWinRPad, f, "ig_pad_r")
			grabInt(call, reWinBPad, f, "ig_pad_b")
			grabInt(call, reWinTPad, f, "ig_pad_t")
			if ax := reWinXalign.FindStringSubmatch(call); ax != nil {
				f["ig_align_x"] = ax[1]
			}
			if ay := reWinYalign.FindStringSubmatch(call); ay != nil {
				f["ig_align_y"] = ay[1]
			}
			if wf := reWhatFont.FindStringSubmatch(call); wf != nil {
				f["ig_font"] = baseName(wf[1])
			}
			grabInt(call, reWhatSize, f, "ig_size")
			if sh := reWhatShadow.FindStringSubmatch(call); sh != nil {
				f["ig_shadow"] = sh[1] + "," + sh[2]
			}
			if sc := reWhatShadowColor.FindStringSubmatch(call); sc != nil {
				f["ig_shadow_color"] = colorA(sc[1])
			}
			if _, ok := f["ig_pad_t"]; !ok {
				if v, ok := f["textbox_pad_t"]; ok {
					f["ig_pad_t"] = v
				} else if v, ok := f["textbox_pad_y"]; ok {
					f["ig_pad_t"] = v
				}
			}
			return
		}
	}
}

func extractSideImages(p *IrProgram, f map[string]string) {
	for _, s := range p.Strings {
		idx := 0
		for {
			i := strings.Index(s[idx:], "Character(")
			if i < 0 {
				break
			}
			i += idx
			open := i + len("Character")
			call := balancedCall(s, open)
			idx = i + 1
			si := strings.Index(call, "show_side_image")
			if si < 0 {
				continue
			}
			cs := strings.Index(call[si:], "ConditionSwitch")
			if cs < 0 {
				continue
			}
			cs += si
			nameM := reCharFirstName.FindStringSubmatch(call)
			if nameM == nil {
				continue
			}
			name := nameM[1]
			csOpen := strings.IndexByte(call[cs:], '(')
			if csOpen < 0 {
				continue
			}
			sw := balancedCall(call, cs+csOpen)
			toks := reQuoted.FindAllStringSubmatch(sw, -1)
			var pairs strings.Builder
			for k := 0; k+1 < len(toks); k += 2 {
				cond := toks[k][1]
				img := toks[k+1][1]
				exprId := p.CompileExpr(cond)
				if exprId < 0 {
					continue
				}
				if pairs.Len() > 0 {
					pairs.WriteByte('|')
				}
				pairs.WriteString(strconv.Itoa(exprId))
				pairs.WriteByte(':')
				pairs.WriteString(baseName(img))
			}
			if pairs.Len() == 0 {
				continue
			}
			xa, ya := "0", "0"
			if ax := reXalign.FindStringSubmatch(sw); ax != nil {
				xa = ax[1]
			}
			if ay := reYalign.FindStringSubmatch(sw); ay != nil {
				ya = ay[1]
			}
			f["side_image."+name] = xa + "," + ya + ";" + pairs.String()
		}
	}
}

func extractTwoWindow(lines []string, p *IrProgram, f map[string]string) {
	var names []string
	for _, s := range p.Strings {
		idx := 0
		for {
			i := strings.Index(s[idx:], "Character(")
			if i < 0 {
				break
			}
			i += idx
			open := i + len("Character")
			call := balancedCall(s, open)
			idx = i + 1
			if !reTwoWindow.MatchString(call) {
				continue
			}
			if nameM := reCharFirstName.FindStringSubmatch(call); nameM != nil && !containsStr(names, nameM[1]) {
				names = append(names, nameM[1])
			}
		}
	}
	if len(names) > 0 {
		f["two_window_names"] = strings.Join(names, "|")
	}
	for _, ln := range lines {
		grab(ln, reWhoBg, f, "who_bg", fmtName)
		grabInt(ln, reWhoXpos, f, "who_xpos")
		grabInt(ln, reWhoYpos, f, "who_ypos")
		grabInt(ln, reWhoLpad, f, "who_lpad")
		grabInt(ln, reWhoTpad, f, "who_tpad")
		grabFloat(ln, reWhoXanchor, f, "who_xanchor")
		grabFloat(ln, reWhoYanchor, f, "who_yanchor")
	}
}

func matchFrame(ln string, f map[string]string) {
	m := reWindowFrame.FindStringSubmatch(ln)
	if m == nil {
		return
	}
	f["textbox_bg"] = baseName(m[1])
	delete(f, "textbox_color")
	if m[4] != "" {
		f["textbox_frame"] = m[2] + "," + m[3] + "," + m[4] + "," + m[5]
	} else {
		f["textbox_frame"] = m[2] + "," + m[3] + "," + m[2] + "," + m[3]
	}
}

func matchChoiceFrame(ln, which string, f map[string]string, imgKey, frameKey string) {
	re := regexp.MustCompile(`style\.menu_choice_button\.` + which + `\s*=\s*Frame\(\s*(?:im\.Image\(\s*)?["']([^"']+)["']\)?\s*,\s*(\d+)\s*,\s*(\d+)(?:\s*,\s*(\d+)\s*,\s*(\d+))?`)
	if m := re.FindStringSubmatch(ln); m != nil {
		f[imgKey] = baseName(m[1])
		if m[4] != "" {
			f[frameKey] = m[2] + "," + m[3] + "," + m[4] + "," + m[5]
		} else {
			f[frameKey] = m[2] + "," + m[3] + "," + m[2] + "," + m[3]
		}
		return
	}
	reIm := regexp.MustCompile(`style\.menu_choice_button\.` + which + `\s*=\s*(?:Image\(\s*)?["']([^"']+)["']`)
	if im := reIm.FindStringSubmatch(ln); im != nil {
		f[imgKey] = baseName(im[1])
	}
}

func matchWindowColor(ln string, f map[string]string) {
	col := ""
	if m := reWindowColorHex.FindStringSubmatch(ln); m != nil {
		col = colorA(m[1])
	}
	if col == "" {
		if m := reWindowColorSolid.FindStringSubmatch(ln); m != nil {
			r, _ := strconv.Atoi(m[1])
			g, _ := strconv.Atoi(m[2])
			b, _ := strconv.Atoi(m[3])
			a := 255
			if m[4] != "" {
				a, _ = strconv.Atoi(m[4])
			}
			col = fmt.Sprintf("#%02x%02x%02x%02x", r&255, g&255, b&255, a&255)
		}
	}
	if col == "" {
		return
	}
	f["textbox_color"] = col
	delete(f, "textbox_bg")
	delete(f, "textbox_frame")
}

func matchScriptVersion(ln string, f map[string]string) {
	if reScriptVer.MatchString(ln) {
		f["text_advance"] = "float"
	}
}

func matchYmargin(ln string, f map[string]string) {
	if m := reYmargin.FindStringSubmatch(ln); m != nil {
		f["textbox_margin_t"] = m[1]
		f["textbox_margin_b"] = m[1]
	}
}

func matchCharColor(ln string, f map[string]string) {
	m := reCharColor.FindStringSubmatch(ln)
	if m == nil {
		return
	}
	col := color(m[2])
	if col != "" {
		f["char_color."+m[1]] = col
	}
}

func matchThemeColor(ln, name string, f map[string]string, key string) {
	reT := regexp.MustCompile(`^\s*` + name + `\s*=\s*\(\s*(\d+)\s*,\s*(\d+)\s*,\s*(\d+)\s*(?:,\s*(\d+)\s*)?\)`)
	if t := reT.FindStringSubmatch(ln); t != nil {
		r, _ := strconv.Atoi(t[1])
		g, _ := strconv.Atoi(t[2])
		b, _ := strconv.Atoi(t[3])
		a := 255
		if t[4] != "" {
			a, _ = strconv.Atoi(t[4])
		}
		f[key] = fmt.Sprintf("#%02x%02x%02x%02x", r&255, g&255, b&255, a&255)
		return
	}
	reH := regexp.MustCompile(`^\s*` + name + `\s*=\s*["'](#[0-9A-Fa-f]{3,8})["']`)
	if h := reH.FindStringSubmatch(ln); h != nil {
		f[key] = colorA(h[1])
	}
}

const fmtRaw, fmtName, fmtColor, fmtColorA = 0, 1, 2, 3

func grab(ln string, rx *regexp.Regexp, f map[string]string, key string, fmtKind int) {
	m := rx.FindStringSubmatch(ln)
	if m == nil {
		return
	}
	v := strings.TrimSpace(m[1])
	switch fmtKind {
	case fmtName:
		v = baseName(unq(v))
	case fmtColor:
		v = color(v)
	case fmtColorA:
		v = colorA(v)
	}
	if v != "" {
		f[key] = v
	}
}

func matchTwo(ln string, rx *regexp.Regexp, f map[string]string, k1, k2 string) {
	m := rx.FindStringSubmatch(ln)
	if m == nil {
		return
	}
	f[k1] = strings.TrimSpace(m[1])
	f[k2] = strings.TrimSpace(m[2])
}

func matchTwoJoin(ln string, rx *regexp.Regexp, f map[string]string, key string) {
	m := rx.FindStringSubmatch(ln)
	if m != nil {
		f[key] = strings.TrimSpace(m[1]) + "," + strings.TrimSpace(m[2])
	}
}

func grabInt(s string, rx *regexp.Regexp, f map[string]string, key string) {
	if m := rx.FindStringSubmatch(s); m != nil {
		f[key] = m[1]
	}
}
func grabFloat(s string, rx *regexp.Regexp, f map[string]string, key string) {
	if m := rx.FindStringSubmatch(s); m != nil {
		f[key] = m[1]
	}
}

func unq(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimRight(s, ",) ")
	if len(s) >= 2 && (s[0] == '"' || s[0] == '\'') && s[len(s)-1] == s[0] {
		return s[1 : len(s)-1]
	}
	return s
}

func color(v string) string {
	v = unq(v)
	m := reHash.FindStringSubmatch(v)
	if m == nil {
		return ""
	}
	h := m[1]
	if len(h) == 3 {
		h = string([]byte{h[0], h[0], h[1], h[1], h[2], h[2]})
	}
	if len(h) >= 6 {
		return "#" + strings.ToLower(h[:6])
	}
	return ""
}

func colorA(v string) string {
	v = unq(v)
	m := reHash.FindStringSubmatch(v)
	if m == nil {
		return ""
	}
	h := strings.ToLower(m[1])
	if len(h) == 3 || len(h) == 4 {
		var sb strings.Builder
		for i := 0; i < len(h); i++ {
			sb.WriteByte(h[i])
			sb.WriteByte(h[i])
		}
		h = sb.String()
	}
	if len(h) == 6 || len(h) == 8 {
		return "#" + h
	}
	return ""
}

func baseName(p string) string {
	if p == "" {
		return p
	}
	p = strings.ReplaceAll(p, `\`, `/`)
	return path.Base(p)
}

func containsStr(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}
func removeStr(list []string, s string) []string {
	out := list[:0:0]
	for _, x := range list {
		if x != s {
			out = append(out, x)
		}
	}
	return append([]string{}, out...)
}

var (
	reWindowTitle       = regexp.MustCompile(`config\.window_title\s*=\s*u?["']([^"']*)["']`)
	reMainMenuMusic     = regexp.MustCompile(`config\.main_menu_music\s*=\s*u?["']([^"']*)["']`)
	reThemeCall         = regexp.MustCompile(`\btheme\.\w+\s*\(`)
	reRoundedWindow     = regexp.MustCompile(`rounded_window\s*=\s*(True|False)`)
	reScreenW           = regexp.MustCompile(`config\.screen_width\s*=\s*(\d+)`)
	reScreenH           = regexp.MustCompile(`config\.screen_height\s*=\s*(\d+)`)
	reThumbW            = regexp.MustCompile(`config\.thumbnail_width\s*=\s*(\d+)`)
	reThumbH            = regexp.MustCompile(`config\.thumbnail_height\s*=\s*(\d+)`)
	reTextCps           = regexp.MustCompile(`config\.default_text_cps\s*=\s*(\d+)`)
	reGuiInit           = regexp.MustCompile(`gui\.init\s*\(\s*(\d+)\s*,\s*(\d+)\s*\)`)
	reStyleFont         = regexp.MustCompile(`style\.(?:default|say_dialogue)\.font\s*=\s*(.+)`)
	reStyleSize         = regexp.MustCompile(`style\.(?:default|say_dialogue)\.size\s*=\s*(\d+)`)
	reStyleColor        = regexp.MustCompile(`style\.(?:default|say_dialogue)\.color\s*=\s*(.+)`)
	reDropShadow        = regexp.MustCompile(`style\.(?:default|say_dialogue)\.drop_shadow\s*=\s*\(\s*(-?\d+)\s*,\s*(-?\d+)\s*\)`)
	reDropShadowColor   = regexp.MustCompile(`style\.(?:default|say_dialogue)\.drop_shadow_color\s*=\s*(.+)`)
	reSayLabelSize      = regexp.MustCompile(`style\.say_label\.size\s*=\s*(\d+)`)
	reSayLabelColor     = regexp.MustCompile(`style\.say_label\.color\s*=\s*(.+)`)
	reWindowBgImg       = regexp.MustCompile(`style\.(?:say_)?window\.background\s*=\s*["']([^"'#][^"']*)["']\s*$`)
	reBottomMargin      = regexp.MustCompile(`style\.(?:say_)?window\.bottom_margin\s*=\s*(\d+)`)
	reTopMargin         = regexp.MustCompile(`style\.(?:say_)?window\.top_margin\s*=\s*(\d+)`)
	reXpadding          = regexp.MustCompile(`style\.(?:say_)?window\.xpadding\s*=\s*(\d+)`)
	reYpadding          = regexp.MustCompile(`style\.(?:say_)?window\.ypadding\s*=\s*(\d+)`)
	reLeftPad           = regexp.MustCompile(`style\.(?:say_)?window\.left_padding\s*=\s*(\d+)`)
	reRightPad          = regexp.MustCompile(`style\.(?:say_)?window\.right_padding\s*=\s*(\d+)`)
	reTopPad            = regexp.MustCompile(`style\.(?:say_)?window\.top_padding\s*=\s*(\d+)`)
	reBottomPad         = regexp.MustCompile(`style\.(?:say_)?window\.bottom_padding\s*=\s*(\d+)`)
	reXmargin           = regexp.MustCompile(`style\.(?:say_)?window\.xmargin\s*=\s*(\d+)`)
	reYmin              = regexp.MustCompile(`style\.(?:say_)?window\.yminimum\s*=\s*(\d+)`)
	reCtcAnim           = regexp.MustCompile(`ctc\s*=\s*anim\.\w+\(\s*["']([^"']+)["']`)
	reGuiCtc            = regexp.MustCompile(`gui\.ctc\s*=\s*["']([^"']+)["']`)
	reCtcXpos           = regexp.MustCompile(`ctc=\s*anim\.\w+\([^)]*\bxpos\s*=\s*(\d+)`)
	reCtcYpos           = regexp.MustCompile(`ctc=\s*anim\.\w+\([^)]*\bypos\s*=\s*(\d+)`)
	reCtcXanchor        = regexp.MustCompile(`ctc=\s*anim\.\w+\([^)]*\bxanchor\s*=\s*(\d+)`)
	reCtcYanchor        = regexp.MustCompile(`ctc=\s*anim\.\w+\([^)]*\byanchor\s*=\s*(\d+)`)
	reCtcFixed          = regexp.MustCompile(`ctc_position\s*=\s*["']fixed["']`)
	reMmRoot            = regexp.MustCompile(`\bmm_root\s*=\s*["']([^"']+)["']`)
	reGmRoot            = regexp.MustCompile(`\bgm_root\s*=\s*["']([^"']+)["']`)
	reTextSizeKw        = regexp.MustCompile(`^\s*text_size\s*=\s*(\d+)`)
	reMmFrameNone       = regexp.MustCompile(`\bmm_menu_frame\.set_parent\s*\(\s*style\.default\s*\)|\bmm_menu_frame\.background\s*=\s*None`)
	reImageButton       = regexp.MustCompile(`^\s*["']([^"']+)["']\s*:\s*\(\s*["']([^"']+\.png)["']\s*,\s*["']([^"']+\.png)["'](?:\s*,\s*["']([^"']+\.png)["'])?(?:\s*,\s*["']([^"']+\.png)["'])?(?:\s*,\s*["']([^"']+\.png)["'])?`)
	reNvlBg             = regexp.MustCompile(`style\.nvl_window\.background\s*=\s*["'](#[0-9A-Fa-f]{3,8})["']`)
	reNvlXpad           = regexp.MustCompile(`style\.nvl_window\.xpadding\s*=\s*(\d+)`)
	reNvlYpad           = regexp.MustCompile(`style\.nvl_window\.ypadding\s*=\s*(\d+)`)
	reNvlSpacing        = regexp.MustCompile(`style\.nvl_vbox\.box_spacing\s*=\s*(\d+)`)
	reSayVboxSpacing    = regexp.MustCompile(`style\.say_vbox\.spacing\s*=\s*(\d+)`)
	reSayLabelBoldFalse = regexp.MustCompile(`style\.say_label\.bold\s*=\s*False`)
	reSayLabelBoldTrue  = regexp.MustCompile(`style\.say_label\.bold\s*=\s*True`)
	reSlotTextSize      = regexp.MustCompile(`style\.file_picker_text\.size\s*=\s*(\d+)`)
	reSlotTextX         = regexp.MustCompile(`style\.file_picker_text_window\.xpos\s*=\s*(\d+)`)
	reSlotTextY         = regexp.MustCompile(`style\.file_picker_text_window\.ypos\s*=\s*(\d+)`)
	reSlotSsX           = regexp.MustCompile(`style\.file_picker_ss_window\.xpos\s*=\s*(\d+)`)
	reSlotSsY           = regexp.MustCompile(`style\.file_picker_ss_window\.ypos\s*=\s*(\d+)`)
	reSlotTextColor     = regexp.MustCompile(`style\.file_picker_text\.color\s*=\s*["'](#[0-9A-Fa-f]{3,8})["']`)
	reChoiceSize        = regexp.MustCompile(`style\.menu_choice\.size\s*=\s*(\d+)`)
	reChoiceXmin        = regexp.MustCompile(`style\.menu_choice_button\.xminimum\s*=\s*(\d+)`)
	reChoiceYmin        = regexp.MustCompile(`style\.menu_choice_button\.yminimum\s*=\s*(\d+)`)
	reChoiceMargin      = regexp.MustCompile(`style\.menu_choice_button\.top_margin\s*=\s*(\d+)`)
	reMenuYalign        = regexp.MustCompile(`style\.menu_window\.yalign\s*=\s*([0-9]*\.?[0-9]+)`)
	reGuiTextSize       = regexp.MustCompile(`gui\.text_size\s*=\s*(\d+)`)
	reGuiTextFont       = regexp.MustCompile(`gui\.text_font\s*=\s*(.+)`)
	reGuiTextColor      = regexp.MustCompile(`gui\.(?:dialogue_)?text_color\s*=\s*(.+)`)
	reGuiNameSize       = regexp.MustCompile(`gui\.name_text_size\s*=\s*(\d+)`)
	reGuiTextboxH       = regexp.MustCompile(`gui\.textbox_height\s*=\s*(\d+)`)
	reHelpOn            = regexp.MustCompile(`config\.help\s*=\s*["'][^"']`)
	reWinBg             = regexp.MustCompile(`window_background\s*=\s*(?:Image|Frame)\(\s*["']([^"']+)["']`)
	reWinLPad           = regexp.MustCompile(`window_left_padding\s*=\s*(\d+)`)
	reWinRPad           = regexp.MustCompile(`window_right_padding\s*=\s*(\d+)`)
	reWinBPad           = regexp.MustCompile(`window_bottom_padding\s*=\s*(\d+)`)
	reWinTPad           = regexp.MustCompile(`window_top_padding\s*=\s*(\d+)`)
	reWinXalign         = regexp.MustCompile(`window_background\s*=\s*(?:Image|Frame)\([^)]*\bxalign\s*=\s*([0-9]*\.?[0-9]+)`)
	reWinYalign         = regexp.MustCompile(`window_background\s*=\s*(?:Image|Frame)\([^)]*\byalign\s*=\s*([0-9]*\.?[0-9]+)`)
	reWhatFont          = regexp.MustCompile(`what_font\s*=\s*["']([^"']+)["']`)
	reWhatSize          = regexp.MustCompile(`what_size\s*=\s*(\d+)`)
	reWhatShadow        = regexp.MustCompile(`what_drop_shadow\s*=\s*\(\s*(-?\d+)\s*,\s*(-?\d+)\s*\)`)
	reWhatShadowColor   = regexp.MustCompile(`what_drop_shadow_color\s*=\s*["'](#[0-9A-Fa-f]{3,8})["']`)
	reCharFirstName     = regexp.MustCompile(`^\(\s*u?["']([^"']+)["']`)
	reQuoted            = regexp.MustCompile(`["']([^"']*)["']`)
	reXalign            = regexp.MustCompile(`\bxalign\s*=\s*([0-9]*\.?[0-9]+)`)
	reYalign            = regexp.MustCompile(`\byalign\s*=\s*([0-9]*\.?[0-9]+)`)
	reTwoWindow         = regexp.MustCompile(`show_two_window\s*=\s*True`)
	reWhoBg             = regexp.MustCompile(`style\.say_who_window\.background\s*=\s*(.+)`)
	reWhoXpos           = regexp.MustCompile(`style\.say_who_window\.xpos\s*=\s*(-?\d+)`)
	reWhoYpos           = regexp.MustCompile(`style\.say_who_window\.ypos\s*=\s*(-?\d+)`)
	reWhoLpad           = regexp.MustCompile(`style\.say_who_window\.left_padding\s*=\s*(\d+)`)
	reWhoTpad           = regexp.MustCompile(`style\.say_who_window\.top_padding\s*=\s*(\d+)`)
	reWhoXanchor        = regexp.MustCompile(`style\.say_who_window\.xanchor\s*=\s*([0-9]*\.?[0-9]+)`)
	reWhoYanchor        = regexp.MustCompile(`style\.say_who_window\.yanchor\s*=\s*([0-9]*\.?[0-9]+)`)
	reWindowFrame       = regexp.MustCompile(`style\.(?:say_)?window\.background\s*=\s*Frame\(\s*(?:im\.Image\(\s*)?["']([^"']+)["']\)?\s*,\s*(\d+)\s*,\s*(\d+)(?:\s*,\s*(\d+)\s*,\s*(\d+))?`)
	reWindowColorHex    = regexp.MustCompile(`style\.(?:say_)?window\.background\s*=\s*["'](#[0-9A-Fa-f]{3,8})["']`)
	reWindowColorSolid  = regexp.MustCompile(`style\.(?:say_)?window\.background\s*=\s*Solid\(\s*\(\s*(\d+)\s*,\s*(\d+)\s*,\s*(\d+)\s*(?:,\s*(\d+)\s*)?\)`)
	reScriptVer         = regexp.MustCompile(`config\.script_version\s*=\s*\(?\s*(\d+)\s*,\s*(\d+)`)
	reYmargin           = regexp.MustCompile(`style\.(?:say_)?window\.ymargin\s*=\s*(\d+)`)
	reCharColor         = regexp.MustCompile(`=\s*Character\(\s*["']([^"']+)["'][^)]*\bcolor\s*=\s*["'](#[0-9A-Fa-f]{3,8})["']`)
	reHash              = regexp.MustCompile(`^#([0-9A-Fa-f]{3,8})$`)
)
