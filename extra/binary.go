package extra

import (
	"log"

	. "github.com/strickyak/tcl67/tcl"
)

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

func cmdBinaryFormat(fr *Frame, argv []T) T {
	f, args := Arg1v(argv)
	var bb []byte

	for _, c := range f.String() {
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
			log.Panicf("Bad format char %d in cmdBinaryFormat", c)
		}
	}

	return MkString(string(bb))
}

var binaryEnsemble = []EnsembleItem{
	EnsembleItem{Name: "format", Cmd: cmdBinaryFormat},
	EnsembleItem{Name: "explode", Cmd: cmdBinaryExplode},
	EnsembleItem{Name: "implode", Cmd: cmdBinaryImplode},
}

func init() {
	if Safes == nil {
		Safes = make(map[string]Command, 333)
	}

	Safes["binary"] = MkEnsemble(binaryEnsemble)
}
