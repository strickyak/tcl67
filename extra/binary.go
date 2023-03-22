package extra

import (
	"log"

	. "github.com/strickyak/tcl67/tcl"
)

func cmdBinarySplit(fr *Frame, argv []T) T {
	a, sz := Arg2(argv)
	var z []T
	bb := []byte(a.String())
	size := sz.Int()
	AssertGT(size, 0)

	for len(bb) > 0 {
		lenbb := int64(len(bb))
		n := size
		if lenbb < size {
			n = lenbb
		}
		// log.Printf("=== lenbb=%d size=%d n=%d bb=%#v", lenbb, size, n, bb)
		z = append(z, MkString(string(bb[:n])))
		// log.Printf("=== z=%#v", z)
		bb = bb[n:]
		// log.Printf("=== bb=%#v", bb)
	}

	return MkList(z)
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
	{Name: "split", Cmd: cmdBinarySplit},
	{Name: "format", Cmd: cmdBinaryFormat},
	{Name: "explode", Cmd: cmdBinaryExplode},
	{Name: "implode", Cmd: cmdBinaryImplode},
}

func init() {
	if Safes == nil {
		Safes = make(map[string]Command, 333)
	}

	Safes["binary"] = MkEnsemble(binaryEnsemble)
}
