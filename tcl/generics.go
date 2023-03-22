package tcl

import (
	"fmt"
	"log"
	"strings"
)

type NUMBER interface {
	byte | rune | int | int64 | float64
}

func Str(a any) string {
	return fmt.Sprintf("%v", a)
}

func StrEach(vec []any) string {
	var b strings.Builder
	b.WriteString("[ ")
	for _, e := range vec {
		b.WriteString(Str(e))
		b.WriteString(", ")
	}
	b.WriteString("]")
	return b.String()
}

func GiveInfo(extra ...any) string {
	if len(extra) == 0 {
		return ""
	}
	return fmt.Sprintf(" info: %s", StrEach(extra))
}

func CheckEQ[N NUMBER](a, b N, extra ...any) {
	if !(a == b) {
		log.Panicf("CHECK FAILS: (%s) EQ (%s) %s", Str(a), Str(b), GiveInfo(extra))
	}
}

func CheckNE[N NUMBER](a, b N, extra ...any) {
	if !(a != b) {
		log.Panicf("CHECK FAILS: (%s) NE (%s) %s", Str(a), Str(b), GiveInfo(extra))
	}
}

func CheckLE[N NUMBER](a, b N, extra ...any) {
	if !(a <= b) {
		log.Panicf("CHECK FAILS: (%s) LE (%s) %s", Str(a), Str(b), GiveInfo(extra))
	}
}

func CheckLT[N NUMBER](a, b N, extra ...any) {
	if !(a < b) {
		log.Panicf("CHECK FAILS: (%s) LT (%s) %s", Str(a), Str(b), GiveInfo(extra))
	}
}

func CheckGE[N NUMBER](a, b N, extra ...any) {
	if !(a >= b) {
		log.Panicf("CHECK FAILS: (%s) GE (%s) %s", Str(a), Str(b), GiveInfo(extra))
	}
}

func CheckGT[N NUMBER](a, b N, extra ...any) {
	if !(a > b) {
		log.Panicf("CHECK FAILS: (%s) GT (%s) %s", Str(a), Str(b), GiveInfo(extra))
	}
}
