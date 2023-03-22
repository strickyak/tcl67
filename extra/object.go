package extra

import (
	. "github.com/strickyak/tcl67/tcl"
)

type Obj struct {
	Label   string
	Thing   any
	Methods []EnsembleItem
}

func MkObj(label string, thing any, methods []EnsembleItem) *Obj {
	return &Obj{
		Label:   label,
		Thing:   thing,
		Methods: methods,
	}
}

// Obj implements T

func (t Obj) String() string {
	return t.Label
}
func (t Obj) ListElementString() string {
	return t.String()
}
func (t Obj) IsQuickString() bool {
	return false
}
func (t Obj) IsQuickList() bool {
	return false
}
func (t Obj) IsQuickHash() bool {
	return false
}
func (t Obj) Bool() bool {
	return false
}
func (t Obj) IsEmpty() bool {
	return false
}
func (t Obj) Float() float64 {
	panic("obj is not a Float")
}
func (t Obj) Int() int64 {
	panic("obj is not an Int")
}
func (t Obj) Uint() uint64 {
	panic("obj is not a Uint")
}
func (t Obj) IsPreservedByList() bool { return false }
func (t Obj) IsQuickInt() bool        { return false }
func (t Obj) IsQuickNumber() bool     { return false }
func (t Obj) List() []T {
	return []T{t}
}
func (t Obj) HeadTail() (hd, tl T) {
	return MkList(t.List()).HeadTail()
}
func (t Obj) Hash() Hash {
	panic(" is not a Hash")
}
func (t Obj) GetAt(key T) T {
	panic("Obj is not a Hash")
}
func (t Obj) PutAt(value T, key T) {
	panic("Obj is not a Hash")
}
func (t Obj) EvalSeq(fr *Frame) T {
	panic("cannot EvalSeq an Obj")
}
func (t Obj) EvalExpr(fr *Frame) T {
	panic("cannot EvalExpr an Obj")
}
func (t Obj) Apply(fr *Frame, args []T) T {
	var newArgs []T
	if len(args) < 1 {
		newArgs = append(newArgs, args[0])
	} else {
		newArgs = append(newArgs, args[0], args[1])
		newArgs = append(newArgs, args[0])     // insert [0] again as self
		newArgs = append(newArgs, args[2:]...) // rest of args
	}

	return MkEnsemble(t.Methods)(fr, newArgs)
}
