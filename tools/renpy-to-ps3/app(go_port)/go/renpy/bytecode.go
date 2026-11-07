package renpy

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

var rbcMagic = []byte("RPYB")

func WriteBytecode(p *IrProgram) []byte {
	var buf bytes.Buffer
	buf.Write(rbcMagic)
	_ = binary.Write(&buf, binary.LittleEndian, uint32(len(p.Code)))
	_ = binary.Write(&buf, binary.LittleEndian, uint32(len(p.Strings)))
	_ = binary.Write(&buf, binary.LittleEndian, uint32(len(p.Labels)))
	entry := uint32(0)
	if st, ok := p.Labels["start"]; ok {
		entry = uint32(st)
	}
	_ = binary.Write(&buf, binary.LittleEndian, entry)

	for _, ins := range p.Code {
		buf.WriteByte(byte(ins.Op))
		_ = binary.Write(&buf, binary.LittleEndian, int32(ins.A))
		_ = binary.Write(&buf, binary.LittleEndian, int32(ins.B))
		_ = binary.Write(&buf, binary.LittleEndian, int32(ins.C))
	}
	for _, s := range p.Strings {
		writeStr(&buf, s)
	}
	for _, pair := range p.LabelPairs() {
		writeStr(&buf, pair[0].(string))
		_ = binary.Write(&buf, binary.LittleEndian, uint32(pair[1].(int)))
	}

	_ = binary.Write(&buf, binary.LittleEndian, uint32(len(p.Exprs)))
	for _, ep := range p.Exprs {
		_ = binary.Write(&buf, binary.LittleEndian, uint32(len(ep.Ops)))
		for _, e := range ep.Ops {
			buf.WriteByte(byte(e.Op))
			_ = binary.Write(&buf, binary.LittleEndian, int32(e.Arg))
		}
	}

	_ = binary.Write(&buf, binary.LittleEndian, uint32(len(p.Atls)))
	for _, ap := range p.Atls {
		_ = binary.Write(&buf, binary.LittleEndian, int32(ap.RepeatCount))
		_ = binary.Write(&buf, binary.LittleEndian, uint32(len(ap.Keys)))
		for _, k := range ap.Keys {
			buf.WriteByte(byte(k.Warper))
			_ = binary.Write(&buf, binary.LittleEndian, int32(k.DurMs))
			buf.WriteByte(byte(len(k.Props)))
			for _, pr := range k.Props {
				buf.WriteByte(byte(pr.Prop))
				_ = binary.Write(&buf, binary.LittleEndian, int32(pr.Value))
			}
		}
	}

	_ = binary.Write(&buf, binary.LittleEndian, uint32(len(p.ImageMaps)))
	for _, d := range p.ImageMaps {
		writeStr(&buf, d.Kind)
		writeStr(&buf, d.Ground)
		writeStr(&buf, d.Idle)
		writeStr(&buf, d.Hover)
		writeStr(&buf, d.SelectedIdle)
		writeStr(&buf, d.SelectedHover)
		_ = binary.Write(&buf, binary.LittleEndian, uint32(len(d.Hotspots)))
		for _, h := range d.Hotspots {
			_ = binary.Write(&buf, binary.LittleEndian, int32(h.X0))
			_ = binary.Write(&buf, binary.LittleEndian, int32(h.Y0))
			_ = binary.Write(&buf, binary.LittleEndian, int32(h.X1))
			_ = binary.Write(&buf, binary.LittleEndian, int32(h.Y1))
			writeStr(&buf, h.Name)
		}
	}

	ovs := p.OverlayList()
	_ = binary.Write(&buf, binary.LittleEndian, uint32(len(ovs)))
	for _, ov := range ovs {
		writeStr(&buf, ov.Name)
		_ = binary.Write(&buf, binary.LittleEndian, uint32(len(ov.Widgets)))
		for _, wd := range ov.Widgets {
			buf.WriteByte(byte(wd.Kind))
			_ = binary.Write(&buf, binary.LittleEndian, int32(wd.X))
			_ = binary.Write(&buf, binary.LittleEndian, int32(wd.Y))
			writeStr(&buf, wd.A)
			writeStr(&buf, wd.B)
			writeStr(&buf, wd.Action)
			_ = binary.Write(&buf, binary.LittleEndian, int32(wd.GuardExpr))
		}
	}
	return buf.Bytes()
}

func writeStr(w io.Writer, s string) {
	b := []byte(s)
	_ = binary.Write(w, binary.LittleEndian, uint32(len(b)))
	_, _ = w.Write(b)
}

type DecodedBytecode struct {
	EntryAddr uint32
	Code      []Instr
	Strings   []string
	Labels    map[string]int
	Exprs     []ExprProgram
	Atls      []AtlProgram
}

func ReadBytecode(data []byte) (*DecodedBytecode, error) {
	r := bytes.NewReader(data)
	m := make([]byte, 4)
	if _, err := io.ReadFull(r, m); err != nil {
		return nil, err
	}
	if string(m) != string(rbcMagic) {
		return nil, fmt.Errorf("bad magic (not an .rbc)")
	}
	d := &DecodedBytecode{Labels: map[string]int{}}
	var instrCount, stringCount, labelCount uint32
	if err := binary.Read(r, binary.LittleEndian, &instrCount); err != nil {
		return nil, err
	}
	if err := binary.Read(r, binary.LittleEndian, &stringCount); err != nil {
		return nil, err
	}
	if err := binary.Read(r, binary.LittleEndian, &labelCount); err != nil {
		return nil, err
	}
	if err := binary.Read(r, binary.LittleEndian, &d.EntryAddr); err != nil {
		return nil, err
	}
	for i := uint32(0); i < instrCount; i++ {
		var op byte
		var a, b, c int32
		if err := binary.Read(r, binary.LittleEndian, &op); err != nil {
			return nil, err
		}
		if err := binary.Read(r, binary.LittleEndian, &a); err != nil {
			return nil, err
		}
		if err := binary.Read(r, binary.LittleEndian, &b); err != nil {
			return nil, err
		}
		if err := binary.Read(r, binary.LittleEndian, &c); err != nil {
			return nil, err
		}
		d.Code = append(d.Code, Instr{Op: IrOp(op), A: int(a), B: int(b), C: int(c)})
	}
	for i := uint32(0); i < stringCount; i++ {
		s, err := readStr(r)
		if err != nil {
			return nil, err
		}
		d.Strings = append(d.Strings, s)
	}
	for i := uint32(0); i < labelCount; i++ {
		name, err := readStr(r)
		if err != nil {
			return nil, err
		}
		var addr uint32
		if err := binary.Read(r, binary.LittleEndian, &addr); err != nil {
			return nil, err
		}
		d.Labels[name] = int(addr)
	}
	if r.Len() > 0 {
		var exprCount uint32
		if err := binary.Read(r, binary.LittleEndian, &exprCount); err != nil {
			return nil, err
		}
		for i := uint32(0); i < exprCount; i++ {
			ep := ExprProgram{}
			var opCount uint32
			if err := binary.Read(r, binary.LittleEndian, &opCount); err != nil {
				return nil, err
			}
			for j := uint32(0); j < opCount; j++ {
				var op byte
				var arg int32
				if err := binary.Read(r, binary.LittleEndian, &op); err != nil {
					return nil, err
				}
				if err := binary.Read(r, binary.LittleEndian, &arg); err != nil {
					return nil, err
				}
				ep.Emit(ExprOp(op), int(arg))
			}
			d.Exprs = append(d.Exprs, ep)
		}
	}
	if r.Len() > 0 {
		var atlCount uint32
		if err := binary.Read(r, binary.LittleEndian, &atlCount); err != nil {
			return nil, err
		}
		for i := uint32(0); i < atlCount; i++ {
			ap := AtlProgram{}
			var rc int32
			if err := binary.Read(r, binary.LittleEndian, &rc); err != nil {
				return nil, err
			}
			ap.RepeatCount = int(rc)
			var keyCount uint32
			if err := binary.Read(r, binary.LittleEndian, &keyCount); err != nil {
				return nil, err
			}
			for k := uint32(0); k < keyCount; k++ {
				key := AtlKey{}
				var warper byte
				if err := binary.Read(r, binary.LittleEndian, &warper); err != nil {
					return nil, err
				}
				key.Warper = AtlWarper(warper)
				var dur int32
				if err := binary.Read(r, binary.LittleEndian, &dur); err != nil {
					return nil, err
				}
				key.DurMs = int(dur)
				var pc byte
				if err := binary.Read(r, binary.LittleEndian, &pc); err != nil {
					return nil, err
				}
				for j := 0; j < int(pc); j++ {
					var prop byte
					var val int32
					if err := binary.Read(r, binary.LittleEndian, &prop); err != nil {
						return nil, err
					}
					if err := binary.Read(r, binary.LittleEndian, &val); err != nil {
						return nil, err
					}
					key.Props = append(key.Props, struct {
						Prop  AtlProp
						Value int
					}{AtlProp(prop), int(val)})
				}
				ap.Keys = append(ap.Keys, key)
			}
			d.Atls = append(d.Atls, ap)
		}
	}
	return d, nil
}

func readStr(r io.Reader) (string, error) {
	var ln uint32
	if err := binary.Read(r, binary.LittleEndian, &ln); err != nil {
		return "", err
	}
	b := make([]byte, ln)
	if _, err := io.ReadFull(r, b); err != nil {
		return "", err
	}
	return string(b), nil
}
