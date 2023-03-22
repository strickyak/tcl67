package extra

import (
	"log"

	. "github.com/strickyak/tcl67/tcl"
)

type Box struct {
	Label string
	Thing T
}

func MkBox(label string, thing T) *Box {
	return &Box{
		Label: label,
		Thing: thing,
	}
}

func cmdBox(fr *Frame, argv []T) T {
	label, thing := Arg2(argv)

	return MkBox(label.String(), thing)
}

func cmdUnbox(fr *Frame, argv []T) T {
	a := Arg1(argv)
	b, ok := a.(*Box)
	if !ok {
		log.Panicf("unbox: not a box: %v", a)
	}

	return b.Thing
}

func init() {
	if Safes == nil {
		Safes = make(map[string]Command, 333)
	}

	Safes["box"] = cmdBox
	Safes["unbox"] = cmdUnbox
}

// Box implements T

func (t Box) String() string {
	return t.Label
}
func (t Box) ListElementString() string {
	return t.String()
}
func (t Box) IsQuickString() bool {
	return false
}
func (t Box) IsQuickList() bool {
	return false
}
func (t Box) IsQuickHash() bool {
	return false
}
func (t Box) Bool() bool {
	return false
}
func (t Box) IsEmpty() bool {
	return false
}
func (t Box) Float() float64 {
	panic("box is not a Float")
}
func (t Box) Int() int64 {
	panic("box is not an Int")
}
func (t Box) Uint() uint64 {
	panic("box is not a Uint")
}
func (t Box) IsPreservedByList() bool { return false }
func (t Box) IsQuickInt() bool        { return false }
func (t Box) IsQuickNumber() bool     { return false }
func (t Box) List() []T {
	return []T{t}
}
func (t Box) HeadTail() (hd, tl T) {
	return MkList(t.List()).HeadTail()
}
func (t Box) Hash() Hash {
	panic(" is not a Hash")
}
func (t Box) GetAt(key T) T {
	panic("Box is not a Hash")
}
func (t Box) PutAt(value T, key T) {
	panic("Box is not a Hash")
}
func (t Box) EvalSeq(fr *Frame) T {
	panic("cannot EvalSeq a Box")
}
func (t Box) EvalExpr(fr *Frame) T {
	panic("cannot EvalExpr a Box")
}
func (t Box) Apply(fr *Frame, args []T) T {
	panic("cannot Apply a Box")
}
