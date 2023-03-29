package extra

import (
	"io/ioutil"
	"log"

	. "github.com/strickyak/tcl67/tcl"
)

func cmdBinarySplit(fr *Frame, argv []T) T {
	a, sz := Arg2(argv)
	var z []T
	bb := []byte(a.String())
	size := sz.Int()
	CheckGT(size, 0)

	for len(bb) > 0 {
		lenbb := int64(len(bb))
		n := size
		if lenbb < size {
			n = lenbb
		}
		z = append(z, MkString(string(bb[:n])))
		bb = bb[n:]
	}

	return MkList(z)
}
func cmdBinaryJoin(fr *Frame, argv []T) T {
	args := Arg0v(argv)
	var z []byte
	for _, a := range args {
		for _, b := range a.List() {
			z = append(z, []byte(b.String())...)
		}
	}
	return MkString(string(z))
}

func cmdBinaryReadfile(fr *Frame, argv []T) T {
	name, more := Arg1v(argv)
	contents, err := ioutil.ReadFile(name.String())
	if err != nil {
		log.Panicf("binary readfile: cannot read file %q: %v", name, err)
	}
	switch len(more) {
	case 0:
		break
	case 1: // offset
		contents = contents[more[0].Int():]
	case 2: // offset, size
		contents = contents[more[0].Int():]
		contents = contents[:more[1].Int()]
	}
	return MkString(string(contents))
}

func cmdBinaryWritefile(fr *Frame, argv []T) T {
	name, contents := Arg2(argv)
	err := ioutil.WriteFile(name.String(), []byte(contents.String()), 0777)
	if err != nil {
		log.Panicf("binary writefile: cannot write file %q: %v", name, err)
	}
	return Empty
}

func cmdBinaryExplode(fr *Frame, argv []T) T {
	a := Arg1(argv)
	var z []T
	for _, c := range a.String() {
		z = append(z, MkInt(int64(c)))
	}
	return MkList(z)
}

func cmdBinaryImplode(fr *Frame, argv []T) T {
	a := Arg1(argv)
	var bb []byte
	for _, e := range a.List() {
		bb = append(bb, byte(e.Int()))
	}
	return MkString(string(bb))
}

func cmdBinaryScan(fr *Frame, argv []T) T {
	sT, fT, vars := Arg2v(argv)
	s := sT.String()

	for _, c := range fT.String() {
		switch c {
		case 'c':
			fr.SetVar(vars[0].String(), MkInt(int64(s[0])))
			s = s[1:]
			vars = vars[1:]

		case 'S':
			hi, lo := int64(s[0]), int64(s[1])
			fr.SetVar(vars[0].String(), MkInt((hi<<8)|lo))
			s = s[2:]
			vars = vars[1:]

		default:
			log.Panicf("Bad format char %q in cmdBinaryScan", c)
		}
	}
	return Empty
}

func cmdBinaryFormat(fr *Frame, argv []T) T {
	fT, args := Arg1v(argv)
	var bb []byte

	for _, c := range fT.String() {
		switch c {
		case 'c':
			a := args[0]
			args = args[1:]
			x := uint(a.Int())
			bb = append(bb, byte(x))
		case 'S':
			a := args[0]
			args = args[1:]
			x := uint(a.Int())
			bb = append(bb, byte(x>>8), byte(x))
		default:
			log.Panicf("Bad format char %q in cmdBinaryFormat", c)
		}
	}

	return MkString(string(bb))
}

var binaryEnsemble = []EnsembleItem{
	{Name: "split", Cmd: cmdBinarySplit},
	{Name: "join", Cmd: cmdBinaryJoin},
	{Name: "explode", Cmd: cmdBinaryExplode},
	{Name: "implode", Cmd: cmdBinaryImplode},
	{Name: "readfile", Cmd: cmdBinaryReadfile},
	{Name: "writefile", Cmd: cmdBinaryWritefile},
	{Name: "scan", Cmd: cmdBinaryScan},
	{Name: "format", Cmd: cmdBinaryFormat},
}

func init() {
	if Safes == nil {
		Safes = make(map[string]Command, 333)
	}

	Safes["binary"] = MkEnsemble(binaryEnsemble)
}
