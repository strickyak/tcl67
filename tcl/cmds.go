package tcl

import (
	"bytes"
	. "fmt"
	"log"
	// "net/http"
	"os"
	R "reflect"
	//"runtime/debug"
	"sort"
	"strings"
	"sync"
	"time"
)

// Safes are builtin commands that safe subinterps can call.
// Conventionally these contain no hyphen.
var Safes map[string]Command

// Unsafes are commands that only the trusted, toplevel terp can call.
// Conventionally these contain a hyphen.
var Unsafes map[string]Command

func IfNilArgvThenUsage(argv []T, usage string) {
	if argv == nil {
		panic(Jump{Status: USAGE, Result: MkString(usage)})
	}
}

func Arg0(argv []T) {
	if len(argv) != 1 {
		panic(Sprintf("Expected 0 arguments, but got argv=%s", Showv(argv)))
	}
}

func Arg0v(argv []T) []T {
	if len(argv) < 1 {
		panic(Sprintf("Expected at least 0 arguments, but got argv=%s", Showv(argv)))
	}
	return argv[1:]
}

func Arg1(argv []T) T {
	if len(argv) != 1+1 {
		panic(Sprintf("Expected 1 arguments, but got argv=%s", Showv(argv)))
	}
	return argv[1]
}

func Arg1Usage(argv []T, usage string) T {
	IfNilArgvThenUsage(argv, usage)
	return Arg1(argv)
}

func Arg1v(argv []T) (T, []T) {
	if len(argv) < 1+1 {
		panic(Sprintf("Expected at least 1 argument, but got argv=%s", Showv(argv)))
	}
	return argv[1], argv[2:]
}

func Arg2(argv []T) (T, T) {
	if len(argv) != 2+1 {
		panic(Sprintf("Expected 2 arguments, but got argv=%s", Showv(argv)))
	}
	return argv[1], argv[2]
}

func Arg2v(argv []T) (T, T, []T) {
	if len(argv) < 2+1 {
		panic(Sprintf("Expected at least 2 arguments, but got argv=%s", Showv(argv)))
	}
	return argv[1], argv[2], argv[3:]
}

// RemoveHeadDashArgs removes dash args from argv (only if they are QuickString), returning the dash args and modified argv.  Argv[0] is preserved.
func RemoveHeadDashArgs(argv []T) (dashes []string, newArgv []T) {
	var i int
	for i = 1; len(argv) >= i; i++ {
		if !argv[i].IsQuickString() {
			break
		}
		str := argv[i].String()
		if len(str) > 0 && (str)[0] == '-' {
			continue
		} else {
			break
		}
	}

	if i == 1 {
		// There were no dashes; don't create new slices.
		newArgv = argv
		return
	}

	dashes = make([]string, i-1)
	for j := 1; j < i; j++ {
		dashes[j-1] = argv[j].String()
	}
	newArgv = make([]T, len(argv)-i+1)
	newArgv[0] = argv[0]
	for j := i; j < len(argv); j++ {
		newArgv[j-i+1] = argv[j]
	}
	return
}

// ArgDash2v expects args to be (1) dash arguments (2) two required args (3) possibly some optional args.
func ArgDash2v(argv []T) ([]string, T, T, []T) {
	dashes, newArgv := RemoveHeadDashArgs(argv)
	if len(newArgv) < 2+1 {
		panic(Sprintf("Expected at least 2 arguments (after the %d dash arguments), but got argv=%s", len(dashes), Showv(argv)))
	}
	return dashes, newArgv[1], newArgv[2], newArgv[3:]
}

func ArgDash2vUsage(argv []T, usage string) ([]string, T, T, []T) {
	IfNilArgvThenUsage(argv, usage)
	return ArgDash2v(argv)
}

func Arg3(argv []T) (T, T, T) {
	if len(argv) != 3+1 {
		panic(Sprintf("Expected 3 arguments, but got argv=%s", Showv(argv)))
	}
	return argv[1], argv[2], argv[3]
}
func Arg3Usage(argv []T, usage string) (T, T, T) {
	IfNilArgvThenUsage(argv, usage)
	return Arg3(argv)
}

func Arg3v(argv []T) (T, T, T, []T) {
	if len(argv) < 3+1 {
		panic(Sprintf("Expected at least 3 arguments, but got argv=%s", Showv(argv)))
	}
	return argv[1], argv[2], argv[3], argv[4:]
}

func Arg4(argv []T) (T, T, T, T) {
	if len(argv) != 4+1 {
		panic(Sprintf("Expected 4 arguments, but got argv=%s", Showv(argv)))
	}
	return argv[1], argv[2], argv[3], argv[4]
}

func Arg5(argv []T) (T, T, T, T, T) {
	if len(argv) != 5+1 {
		panic(Sprintf("Expected 5 arguments, but got argv=%s", Showv(argv)))
	}
	return argv[1], argv[2], argv[3], argv[4], argv[5]
}

func Arg6(argv []T) (T, T, T, T, T, T) {
	if len(argv) != 6+1 {
		panic(Sprintf("Expected 6 arguments, but got argv=%s", Showv(argv)))
	}
	return argv[1], argv[2], argv[3], argv[4], argv[5], argv[6]
}

func Arg7(argv []T) (T, T, T, T, T, T, T) {
	if len(argv) != 7+1 {
		panic(Sprintf("Expected 7 arguments, but got argv=%s", Showv(argv)))
	}
	return argv[1], argv[2], argv[3], argv[4], argv[5], argv[6], argv[7]
}

func cmdUsage(fr *Frame, argv []T) T {
	usage := `cmdName -> usageString`
	cmdName := Arg1Usage(argv, usage)

	result := ""

	c := fr.FindCommand(cmdName, false)
	if c != nil {
		func() {
			defer func() {
				r := recover()
				if r != nil {
					switch x := r.(type) {
					case Jump:
						if x.Status == USAGE {
							result = x.Result.String()
						}
					}
				}
			}()
			c(fr, nil) // Call the command with nil args; it may panic Jump USAGE.
		}()

		if result != "" {
			return MkString(Sprintf("*** Usage:  %s %s", cmdName, result))
		}
	}

	panic(Sprintf("Usage not found for command: %q", cmdName))
}

var MustMutex sync.Mutex
var MustSucceeds int64
var MustFails int64

func cmdMust(fr *Frame, argv []T) T {
	xx, yy, rest := Arg2v(argv)
	x := xx.String()
	y := yy.String()

	if x != y {
		MustMutex.Lock()
		MustFails++
		MustMutex.Unlock()

		msg := ""
		for _, e := range rest {
			msg += Sprintf(" ;; %v", SubstStringOrOrig(fr, e.String()))
		}
		panic("FAILED: must: " + Repr(argv) + " ;;;; x=<" + x + "> ;;;; y=<" + y + "> ;;;;" + msg)
	}
	MustMutex.Lock()
	MustSucceeds++
	MustMutex.Unlock()
	return Empty
}

func cmdMustFail(fr *Frame, argv []T) T {
	xx := Arg1(argv)
	var recovered interface{}

	func() { // A scope for defer.

		defer func() {
			recovered = recover()
		}()

		fr.Eval(xx)

	}()

	if recovered == nil {
		MustMutex.Lock()
		MustFails++
		MustMutex.Unlock()
		panic("mustfil but did not fail: " + Repr(argv))
	}

	MustMutex.Lock()
	MustSucceeds++
	MustMutex.Unlock()
	return Empty
}

func cmdIf(fr *Frame, argv []T) T {
	if len(argv) < 3 {
		panic(Sprintf("Too few arguments for if: %#v", argv))
	}
	var cond, yes, no T

	switch len(argv) {
	case 5:
		if argv[3].String() != "else" {
			panic(Sprintf("Expected 'else' at argv[3]: %#v", argv))
		}
		cond, yes, no = argv[1], argv[2], argv[4]
	case 3:
		cond, yes = argv[1], argv[2]
	default:
		panic(Sprintf("Wrong len(argv) for if: %#v", argv))
	}

	if fr.EvalExpr(cond).Bool() {
		return fr.Eval(yes)
	}

	if no != nil {
		return fr.Eval(no)
	}

	return Empty
}

func cmdCase(fr *Frame, argv []T) T {
	// Two possible syntaxes for Tcl 6.7 case command:
	//   (1) case string ?in? patList body ?patList body ...?
	//   (2) case string ?in? {patList body ?patList body ...?}
	topicL, rest := Arg1v(argv)
	topic := topicL.String()

	if len(rest) < 1 {
		panic(Sprintf("Too few arguments for 'case': %#v", argv))
	}
	if rest[0].String() == "in" {
		// ?in? exists; delete it.
		rest = rest[1:]
	}

	if len(rest) == 1 {
		// Case (2).  Expand the one arg into its parts.
		rest = rest[0].List()
	}

	if (len(rest) & 1) == 1 {
		panic(Sprintf("Odd number of items in {patList body} list of stride two: %v", argv))
	}

	var dflt T
	for i := 0; i < len(rest); i += 2 {
		pats := rest[i].List()
		if len(pats) == 1 && pats[0].String() == "default" {
			dflt = rest[i+1]
			continue
		}
		for _, pat := range pats {
			if StringMatch(pat.String(), topic) {
				return fr.Eval(rest[i+1])
			}
		}
	}

	if dflt == nil {
		return Empty
	}
	return fr.Eval(dflt)
}

func nextFormatLetter(s string) (z string, c byte, t R.Kind) {
	// Advance to %
	var i int = 0
	for i < len(s) {
		if s[i] == '%' {
			// Advance to main letter.
			i++
			for i < len(s) {
				switch s[i] {
				case 'v', 'T':
					return s[i+1:], s[i], R.Struct
				case '%':
					return s[i+1:], '%', R.Uint8
				case 't':
					return s[i+1:], s[i], R.Bool
				case 'b', 'c', 'd', 'o', 'x', 'X', 'U':
					return s[i+1:], s[i], R.Int
				case 'e', 'E', 'f', 'F', 'g', 'G': // not 'b'
					return s[i+1:], s[i], R.Float64
				case 's', 'q': // not 'x', 'X'
					return s[i+1:], s[i], R.String
				case 'p':
					return s[i+1:], s[i], R.Ptr
				}
				i++
			}
			return "", 0, R.Invalid
		} else {
			i++
		}
	}
	return "", 0, R.Invalid
}

func cmdFormat(fr *Frame, argv []T) T {
	f, args := Arg1v(argv)
	s := f.String()
	var vals []interface{}
	var i int = 0
	for {
		var c byte
		var k R.Kind
		s, c, k = nextFormatLetter(s)
		if c != '%' {
			if k == R.Invalid {
				break
			}
			if i >= len(args) {
				panic("format: Not enough args")
			}
			switch k {
			case R.Bool:
				vals = append(vals, args[i].Bool())
			case R.Int:
				vals = append(vals, args[i].Int())
			case R.Float64:
				vals = append(vals, args[i].Float())
			case R.String:
				vals = append(vals, args[i].String())
			default:
				log.Panicf("cmdFormat: bad case: %v", k)
			}
			i++
		}
	}
	if i != len(args) {
		panic("format: Too many args")
	}
	return MkString(Sprintf(f.String(), vals...))
}
func cmdScan(fr *Frame, argv []T) T {
	sT, fT, args := Arg2v(argv)
	f := fT.String()
	var ptrs []interface{}
	var i int = 0
	for {
		var c byte
		var k R.Kind
		f, c, k = nextFormatLetter(f)
		if c != '%' {
			if k == R.Invalid {
				break
			}
			if i >= len(args) {
				panic("format: Not enough args")
			}
			switch k {
			case R.Bool:
				ptrs = append(ptrs, new(bool))
			case R.Int:
				ptrs = append(ptrs, new(int64))
			case R.Float64:
				ptrs = append(ptrs, new(float64))
			case R.String:
				ptrs = append(ptrs, new(string))
			default:
				panic("scan: Cannot handle a format")
				//case R.Ptr:
				//case R.Struct:
			}
			i++
		}
	}
	if i != len(args) {
		panic("format: Too many args")
	}
	n, err := Sscanf(sT.String(), fT.String(), ptrs...)
	if err != nil {
		panic(Sprintf("error in Sscanf: %v", err))
	}
	for j, ej := range ptrs {
		if j == n {
			break
		}
		var thing T
		switch t := ej.(type) {
		case *bool:
			thing = MkBool(*t)
		case *int64:
			thing = MkInt(*t)
		case *float64:
			thing = MkFloat(*t)
		case *string:
			thing = MkString(*t)
		default:
			log.Panicf("Bad case in cmdScan: (%T) %#v", ej, ej)
		}
		fr.SetVar(args[j].String(), thing)
	}
	return MkInt(int64(n))
}

func cmdEcho(fr *Frame, argv []T) T {
	args := Arg0v(argv)
	buf := bytes.NewBuffer(nil)
	gap := ""
	for _, a := range args {
		buf.WriteString(gap)
		gap = " "
		buf.WriteString(a.String())
	}
	Println(buf.String())
	return Empty
}

func cmdSay(fr *Frame, argv []T) T {
	args := Arg0v(argv)
	buf := bytes.NewBuffer(nil)
	for _, a := range args {
		buf.WriteString(" say: ")
		buf.WriteString(a.String())
	}
	log.Println(buf.String())
	return MkList(args) // "say" acts like "list"
}

func cmdMacro(fr *Frame, argv []T) T {
	name, aa, body := Arg3(argv)
	nameStr := name.String()
	alist := aa.List()

	astrs := make([]string, len(alist))
	for i, arg := range alist {
		astrs[i] = arg.String()
	}

	_, okCmd := fr.G.Cmds[nameStr]
	if okCmd {
		panic("Command already exists: " + nameStr)
	}

	_, okMacro := fr.G.Macros[nameStr]
	if okMacro {
		panic("Macro already exists: " + nameStr)
	}

	var seq *PSeq = CompileSequence(fr, body.String())

	fr.G.Macros[nameStr] = &MacroNode{
		Args: astrs,
		Body: seq,
	}

	return Empty
}

func cmdProc(fr *Frame, argv []T) T {
	return purifiedProc(fr, argv)
}

/*
func cmdYProc(fr *Frame, argv []T) T {
	return procOrYProc(fr, argv, true)
}
*/

func purifiedProc(fr *Frame, argv []T) T {
	name, aa, body := Arg3(argv)
	nameStr := name.String()
	alist := aa.List()
	astrs := make([]string, len(alist))
	dflts := make([]T, len(alist))
	for i, arg := range alist {
		avec := arg.List()
		var astr string
		switch len(avec) {
		case 1:
			astr = arg.String()
		case 2:
			astr = avec[0].String()
			dflts[i] = avec[1]
		default:
			panic("proc: Formal Parameter llength is not 1 or 2")
		}
		if !IsLocal(astr) {
			panic(Sprintf("Cannot use nonlocal name %q for argument in %s", astr, argv[0]))
		}
		astrs[i] = astr
	}
	n := len(alist)

	compiled := CompileSequence(fr, body.String())

	cmd := func(fr2 *Frame, argv2 []T) (result T) {
		// If generating, not enough happens in this func (as opposed to
		// in the goroutine) to encounter errors.  So this defer/recover is only
		// for the normal, nongenerating case.
		defer func() {
			if r := recover(); r != nil {
				if j, ok := r.(Jump); ok {
					switch j.Status {
					case RETURN:
						result = j.Result
						return
					case BREAK:
						r = ("break command was not inside a loop")
					case CONTINUE:
						r = ("continue command was not inside a loop")
					}
				}
				if rs, ok := r.(string); ok {
					rs = rs + "\n\tin proc " + argv2[0].String()
					// TODO: Require debug level for the args.
					for ai, ae := range argv2[1:] {
						as := ae.String()
						if len(as) > 80 {
							as = as[:80] + "..."
						}
						rs = rs + Sprintf("\n\t\targ:%d = %q", ai, as)
					}
					// TODO: Require debug level for the locals.
					for vk, vv := range fr2.Vars {
						vs := vv.Get().String()
						if len(vs) > 80 {
							vs = vs[:80] + "..."
						}
						rs = rs + Sprintf("\n\t\tlocal:%s = %q", vk, vs)
					}
					r = rs
				}
				panic(r) // Rethrow errors and unknown Status.
			}
		}()

		if argv2 == nil {
			// Debug Data, if invoked with nil argv2.
			return MkList(argv)
		}

		var varargs bool = false
		if len(astrs) > 0 && astrs[len(astrs)-1] == "args" {
			// TODO: Support dflts with varargs.
			varargs = true
			if len(argv2) < n {
				panic(Sprintf("%s %q expects arguments %#v but got %d", argv[0], nameStr, aa, len(argv2)))
			}
		} else {
			// Handle dflts with non-varargs.
			for p := len(argv2); p < n+1; p++ {
				if dflts[p-1] != nil {
					argv2 = append(argv2, dflts[p-1])
				} else {
					break
				}
			}

			if len(argv2) != n+1 {
				panic(Sprintf("%s %q expects arguments %#v but got %d", argv[0], nameStr, aa, len(argv2)))
			}
		}

		fr3 := fr2.NewFrame()
		fr3.DebugName = nameStr

		if varargs {
			for i, arg := range astrs[:len(astrs)-1] {
				fr3.SetVar(arg, argv2[i+1])
			}

			fr3.SetVar("args", MkList(argv2[len(astrs):]))
		} else {
			for i, arg := range astrs {
				fr3.SetVar(arg, argv2[i+1])
			}
		}

		return compiled.Eval(fr3)
	}

	builtin := Safes[nameStr]
	if builtin != nil {
		panic(Sprintf("cannot redefine a builtin: %q", nameStr))
	}

	existingNode := fr.G.Cmds[nameStr]

	if existingNode != nil {
		panic(Sprintf("Name already defined at base level; cannot redefine: %q", nameStr))
	}

	// Install base command.
	node := &CmdNode{
		Fn:   cmd,
		Next: nil,
	}
	fr.G.Cmds[nameStr] = node

	return Empty
}

func cmdSLen(fr *Frame, argv []T) T {
	a := Arg1(argv)
	return MkInt(int64(len(a.String())))
}

func cmdLLen(fr *Frame, argv []T) T {
	a := Arg1(argv)
	return MkInt(int64(len(a.List())))
}

func cmdList(fr *Frame, argv []T) T {
	return MkList(argv[1:])
}

func cmdLIndex(fr *Frame, argv []T) T {
	tlist, ti := Arg2(argv)
	list := tlist.List()
	istr := ti.String()

	var i int64
	if istr == "end" {
		i = (int64)(len(list) - 1)
	} else if len(istr) > 4 && istr[:3] == "end-" {
		panic("unimplemented: lrange ends with 'end-N'")
	} else {
		i = ti.Int()
	}

	if i < 0 || i > int64(len(list)) {
		panic(Sprintf("lindex: bad index: len(list)=%d but i=%d", len(list), i))
	}
	return list[i]
}

func cmdLRange(fr *Frame, argv []T) T {
	tlist, tbegin, tend := Arg3(argv)
	list := tlist.List()
	begin := tbegin.Int()
	endStr := tend.String()

	var end int64
	if endStr == "end" {
		end = (int64)(len(list) - 1)
	} else if len(endStr) > 4 && endStr[:3] == "end-" {
		panic("unimplemented: lrange ends with 'end-N'")
	} else {
		end = tend.Int()
	}

	// Now convert to C++ style end, which points to slot after the last one.
	end++

	if begin < 0 || begin > int64(len(list)) {
		panic(Sprintf("lindex: bad index: len(list)=%d but begin=%d", len(list), begin))
	}
	if end < 0 || end > int64(len(list)) {
		panic(Sprintf("lindex: bad index: len(list)=%d but end=%d", len(list), end))
	}
	if end <= begin {
		return Empty
	}
	n := (int)(end - begin)
	z := make([]T, n)
	for i := 0; i < n; i++ {
		z[i] = list[(int)(begin)+i]
	}
	return MkList(z)
}

type lsorter struct {
	asInt      bool
	asReal     bool
	descending bool
	index      int64
	vec        []T
}

func (o *lsorter) Len() int {
	return len(o.vec)
}
func xor(a, b bool) bool {
	if a {
		return !b
	}
	return b
}
func ith(t T, index int64) T {
	if index < 0 {
		return t
	}
	return t.List()[index]
}
func (o *lsorter) Less(i, j int) bool {
	// Would be more efficient to extract the ith at creation.
	if o.asInt {
		a := ith(o.vec[i], o.index).Int()
		b := ith(o.vec[j], o.index).Int()
		return xor(a < b, o.descending)
	} else if o.asReal {
		a := ith(o.vec[i], o.index).Float()
		b := ith(o.vec[j], o.index).Float()
		return xor(a < b, o.descending)
	} else {
		a := ith(o.vec[i], o.index).String()
		b := ith(o.vec[j], o.index).String()
		return xor(a < b, o.descending)
	}
}
func (o *lsorter) Swap(i, j int) {
	o.vec[i], o.vec[j] = o.vec[j], o.vec[i]
}

func cmdLSort(fr *Frame, argv []T) T {
	// lsort ?options? list
	if len(argv) < 2 {
		panic("lsort needs a list arg")
	}

	vecT := argv[len(argv)-1]
	vec := vecT.List()
	n := len(vec)
	if n <= 1 {
		// Consider a list with 1 or less elements already sorted.
		return vecT
	}

	c := &lsorter{
		asInt:      false,
		asReal:     false,
		descending: false,
		index:      -1,
		vec:        vec,
	}

	opts := argv[1 : len(argv)-1] // Remove cmd name (first) and the list (final).
	for len(opts) > 0 {
		opt := opts[0].String()
		if strings.HasPrefix(opt, "-int") {
			c.asInt = true
			opts = opts[1:]
		} else if strings.HasPrefix(opt, "-r") {
			c.asReal = true
			opts = opts[1:]
		} else if strings.HasPrefix(opt, "-d") {
			c.descending = true
			opts = opts[1:]
		} else if strings.HasPrefix(opt, "-ind") && len(opts) > 1 {
			c.index = opts[1].Int()
			opts = opts[2:]
		} else {
			panic("lsort: bad option")
		}
	}

	// Sort our strings.
	sort.Sort(c)

	// Return the sorted list.
	return MkList(c.vec)
}

func cmdLReverse(fr *Frame, argv []T) T {
	tt := Arg1(argv)
	v := tt.List()
	n := len(v)
	for i := 0; i < n/2; i++ {
		v[i], v[n-i-1] = v[n-i-1], v[i]
	}
	return MkList(v)
}

func cmdSAt(fr *Frame, argv []T) T {
	s, j := Arg2(argv)
	i := j.Int()
	return MkString(s.String()[i : i+1])
}

func cmdForEach(fr *Frame, argv []T) T {
	varLT, list, body := Arg3(argv)
	varL := varLT.List()

	toBreak := false
	toContinue := false

Outer:
	for {
		var hd T
		var tl T
		for _, varT := range varL {
			hd, tl = list.HeadTail()
			if hd == nil {
				// This does leave vars in a slightly skewed state if the stride of varL
				// doesn't fit the data.  So just don't mismatch the lengths.
				// It's not worth the complexity to fix this code.
				break Outer
			}
			list = tl

			fr.SetVar(varT.String(), hd)
		}

		func() {
			defer func() {
				if r := recover(); r != nil {
					if j, ok := r.(Jump); ok {
						switch j.Status {
						case BREAK:
							toBreak = true
							return
						case CONTINUE:
							toContinue = true
							return
						}
					}
					panic(r) // Rethrow errors and unknown Status.
				}
			}()
			fr.Eval(body)
		}()
		if toBreak {
			break
		}
		if toContinue {
			continue
		}
	}

	return Empty
}

func cmdWhile(fr *Frame, argv []T) T {
	cond, body := Arg2(argv)

	toBreak := false
	toContinue := false

	for {
		c := fr.EvalExpr(cond)
		if !c.Bool() {
			break
		}

		func() {
			defer func() {
				if r := recover(); r != nil {
					if j, ok := r.(Jump); ok {
						switch j.Status {
						case BREAK:
							toBreak = true
							return
						case CONTINUE:
							toContinue = true
							return
						}
					}
					panic(r) // Rethrow errors and unknown Status.
				}
			}()
			fr.Eval(body)
		}()
		if toBreak {
			break
		}
		if toContinue {
			continue
		}
	}

	return Empty
}

func cmdCatch(fr *Frame, argv []T) (status T) {
	body, optionalName := Arg1v(argv)
	var varName string
	switch len(optionalName) {
	case 0:
		// Leave varName empty.
	case 1:
		varName = optionalName[0].String()
	default:
		panic("catch: too many args")
	}

	defer func() {
		if r := recover(); r != nil {

			// println(Sprintf("\n\n%%%%%%%%%%%%%% catch: CAUGHT EXCEPTION: %T: %v", r, r))
			// println("%%%%%%%%%%%%%% catch: CAUGHT EXCEPTION PrintStack {")
			// debug.PrintStack()
			// println("%%%%%%%%%%%%%% catch: CAUGHT EXCEPTION PrintStack }\n\n")

			// Handle catching Jump objects.
			if j, ok := r.(Jump); ok {
				if len(varName) > 0 {
					fr.SetVar(varName, j.Result)
				}
				status = MkInt(int64(j.Status))
				return
			}

			if len(varName) > 0 {
				fr.SetVar(varName, MkString(Sprintf("%v", r)))
			}
			status = True
		}
	}()

	z := fr.Eval(body)
	fr.SetVar(varName, z)
	return False
}

var clockEnsemble = []EnsembleItem{
	EnsembleItem{Name: "seconds", Cmd: cmdClockSeconds},
	EnsembleItem{Name: "milliseconds", Cmd: cmdClockMilliseconds},
	EnsembleItem{Name: "microseconds", Cmd: cmdClockMicroseconds},
	EnsembleItem{Name: "format", Cmd: cmdClockFormat},
}

func cmdClockSeconds(fr *Frame, argv []T) T {
	Arg0(argv)
	u := time.Now().Unix()
	return MkInt(int64(u))
}

func cmdClockMilliseconds(fr *Frame, argv []T) T {
	Arg0(argv)
	u := time.Now().UnixNano()
	return MkInt(int64(u / 1000000))
}

func cmdClockMicroseconds(fr *Frame, argv []T) T {
	Arg0(argv)
	u := time.Now().UnixNano()
	return MkInt(int64(u / 1000))
}

func cmdClockFormat(fr *Frame, argv []T) T {
	secsT, restT := Arg1v(argv)
	f := time.UnixDate
	location := time.Local
	for len(restT) >= 2 {
		flagT, valT, moreT := restT[0], restT[1], restT[2:]
		switch flagT.String() {
		case "-format":
			f = valT.String()
		case "-gmt":
			if valT.Bool() {
				location = time.UTC
			}
		default:
			panic("Unknown flag to {clock format}")
		}
		restT = moreT
	}
	if len(restT) > 0 {
		panic("Odd number of args after {clock format}")
	}
	return MkString(time.Unix(0, int64(1000000000*secsT.Float())).In(location).Format(f))
}

func cmdTime(fr *Frame, argv []T) T {
	cmd, rest := Arg1v(argv)
	var n int64
	switch len(rest) {
	case 0:
		n = 1
	case 1:
		n = rest[0].Int()
	default:
		panic("time: too many args")
	}
	start := time.Now().UnixNano()
	var i int64
	for i < n {
		fr.Eval(cmd)
		i++
	}
	finish := time.Now().UnixNano()
	return MkString(Sprintf("%.6f microseconds per iteration", float64(finish-start)/1000.0/float64(n)))
}

func cmdEval(fr *Frame, argv []T) T {
	return EvalOrApplyLists(fr, argv[1:])
}

func cmdGo(fr *Frame, argv []T) T {
	go EvalOrApplyLists(fr, argv[1:])
	return Empty
}

// uplevel requres first arg specifying what level.
// Valid are "#0" (global) or a positive integer (relative).
func cmdUpLevel(fr *Frame, argv []T) T {
	specArg, rest := Arg1v(argv)
	spec := specArg.String()

	// Special case for #0 meaning global.
	if spec == "#0" {
		return EvalOrApplyLists(&fr.G.Fr, rest)
	}

	// Count back number of frames specified.
	level := specArg.Int()
	for i := int64(0); i < level; i++ {
		if fr.Prev != nil {
			fr = fr.Prev
		}
	}
	return EvalOrApplyLists(fr, rest)
}

func EvalOrApplyLists(fr *Frame, lists []T) T {
	if Debug['a'] {
		Say("hello EvalOrApplyLists", Showv(lists))
	}

	///////// I don't think we care about non-lists;  I've never used concat or eval or uplevel with non-lists.
	//
	// Are they already lists?
	areLists := true
	if Debug['z'] {
		Say("areLists := true")
	}
	for _, e := range lists {
		if Debug['z'] {
			Say("areLists ?", e)
		}
		//zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz
		//Should try calling List() on each one.
		//e.IsPreservedByList just means it will be a singleton list.
		//When we are EvalOrApplyLists, don't we really want a list?
		//Or does this break something, vs. joining on space?
		//Shouldn't we break Tcl, if it does?
		//zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz

		if !e.IsPreservedByList() {
			if Debug['z'] {
				Say("areLists := false")
			}
			areLists = false
			break
		}
	}
	if Debug['z'] {
		Say("areLists ->", areLists)
	}

	if areLists {
		cat := ConcatLists(lists)
		if len(cat) == 0 {
			// Because in Tcl, "eval [list]" -> "".
			return Empty
		}
		if Debug['z'] {
			Say("Sending Apply to ", cat[0])
		}
		z := cat[0].Apply(fr, cat)
		if Debug['z'] {
			Say("Sending Apply returns ->", z)
		}
		return z
	}

	buf := bytes.NewBuffer(nil)
	for _, e := range lists {
		buf.WriteString(e.String())
		buf.WriteRune(' ')
	}
	return fr.Eval(MkString(buf.String()))
}

func ConcatLists(lists []T) []T {
	z := make([]T, 0, 4)
	for _, e := range lists {
		z = append(z, e.List()...)
	}
	return z
}

func cmdConcat(fr *Frame, argv []T) T {
	return MkList(ConcatLists(argv[1:]))
}

func cmdGlobal(fr *Frame, argv []T) T {
	gFr := &fr.G.Fr
	for _, a := range argv[1:] {
		aName := a.String()
		fr.DefineUpVar(aName, gFr, aName)
	}
	return Empty
}

func cmdUpVar(fr *Frame, argv []T) T {
	lev, rem, loc := Arg3(argv)
	remName := rem.String()
	locName := loc.String()
	remFr := fr
	if lev.String() == "#0" {
		// Global scope.
		remFr = &fr.G.Fr
	} else {
		level := lev.Int()
		// println(Sprintf("upvar-level=%d [%s]scope=%v globals=%v", 0, remFr.DebugName, remFr.Vars, remFr.G.Fr.Vars))
		for i := 0; i < int(level); i++ {
			remFr = remFr.Prev
			// println(Sprintf("upvar-level=%d [%s]scope=%v globals=%v", i+1, remFr.DebugName, remFr.Vars, remFr.G.Fr.Vars))
		}
	}
	fr.DefineUpVar(locName, remFr, remName)
	// println(Sprintf("upvar-level  defined  [%s]scope=%v globals=%v", remFr.DebugName, remFr.Vars, remFr.G.Fr.Vars))
	return Empty
}

func cmdSet(fr *Frame, argv []T) T {
	target, _ := Arg1v(argv)
	targ := target.String()
	if len(targ) == 0 {
		panic("command 'set' target is empty")
	}
	n := len(targ)
	if targ[n-1] == ')' {
		// Case Subscript:
		i := strings.Index(targ, "(")
		if i < 0 {
			panic("command 'set' target ends with ')' but has no '('")
		}
		if i < 1 {
			panic("command 'set' target is empty before '('")
		}

		name := targ[:i]
		key := targ[i+1 : n-1]
		if len(argv) == 2 {
			h := fr.GetVar(name)
			return h.GetAt(MkString(key))
		}
		if !fr.HasVar(name) {
			fr.SetVar(name, MkHash(nil))
		}
		_, x := Arg2(argv)
		h := fr.GetVar(name)
		h.PutAt(x, MkString(key))
		return x
	}

	// Case No Subscript:
	if len(argv) == 2 {
		// Retrieve value of variable, if 2nd arg is missing.
		name := Arg1(argv)
		return fr.GetVar(name.String())
	}
	name, x := Arg2(argv)
	fr.SetVar(name.String(), x)
	return x
}

func cmdReturn(fr *Frame, argv []T) T {
	var z T = Empty
	if len(argv) == 2 {
		z = argv[1]
	}
	if len(argv) > 2 {
		z = MkList(argv[1:])
	}
	// Jump with status RETURN.
	panic(Jump{Status: RETURN, Result: z})
}

func cmdBreak(fr *Frame, argv []T) T {
	panic(Jump{Status: BREAK}) // Jump with status BREAK.
}

func cmdContinue(fr *Frame, argv []T) T {
	panic(Jump{Status: CONTINUE}) // Jump with status CONTINUE.
}

func cmdHash(fr *Frame, argv []T) T {
	args := Arg0v(argv)
	h := make(Hash)

	// Special case of 1 arg: split it.
	if len(args) == 1 {
		args = args[0].List()
	}

	if len(args)%2 != 0 {
		panic("hash command cannot take odd number of key & value items")
	}

	i := 0
	for i+1 < len(args) {
		h[args[i].String()] = args[i+1]
		i += 2
	}
	return MkHash(h)
}

func cmdHGet(fr *Frame, argv []T) T {
	hash, key := Arg2(argv)
	h := hash.Hash()
	k := key.String()
	value, ok := h[k]
	if !ok {
		panic(Sprintf("Hash does not contain key: %q", k))
	}
	return value
}

func cmdHSet(fr *Frame, argv []T) T {
	hash, key, value := Arg3(argv)
	h := hash.Hash()
	k := key.String()
	h[k] = value
	return value
}

func cmdHDel(fr *Frame, argv []T) T {
	hash, key := Arg2(argv)
	h := hash.Hash()
	k := key.String()
	delete(h, k)
	return Empty
}

func hashKeys(h Hash) []T {
	z := make([]T, 0, len(h))
	for _, k := range SortedKeysOfHash(h) {
		z = append(z, MkString(k))
	}
	return z
}

func cmdHKeys(fr *Frame, argv []T) T {
	hash := Arg1(argv)
	h := hash.Hash()
	return MkList(hashKeys(h))
}

// Tcl requires integers, but our base numeric value is float64.
func cmdIncr(fr *Frame, argv []T) T {
	var varName, delta T
	if len(argv) == 2 {
		varName = Arg1(argv)
		delta = One
	} else {
		varName, delta = Arg2(argv)
	}

	name := varName.String()

	if !fr.HasVar(name) {
		fr.SetVar(name, Zero)
	}
	v := fr.GetVar(name).Float()
	i := delta.Float()
	z := MkFloat(v + i)

	fr.SetVar(name, z)

	return z
}

func cmdAppend(fr *Frame, argv []T) T {
	varName, values := Arg1v(argv)

	name := varName.String()

	if !fr.HasVar(name) {
		fr.SetVar(name, Empty)
	}
	v := fr.GetVar(name)

	i := 0
	n := len(values)

	if n == 0 {
		// We get to return early.
		return v
	}

	buf := bytes.NewBufferString(v.String())

	for i < n {
		buf.WriteString(values[i].String())
		i++
	}

	z := MkString(buf.String())
	fr.SetVar(name, z)
	return z
}

func cmdLAppend(fr *Frame, argv []T) T {
	varName, values := Arg1v(argv)

	name := varName.String()

	if !fr.HasVar(name) {
		fr.SetVar(name, Empty)
	}
	v := fr.GetVar(name).List()
	v = append(v, values...)

	z := MkList(v)
	fr.SetVar(name, z)
	return z
}

func cmdError(fr *Frame, argv []T) T {
	message := Arg1(argv)

	panic(message.String())
}

// Modern Tcl uses "return --code" to throw strange codes,
// but our "return" takes multiple values, so we cannot use it.
// Tcl 6.7 had no way to do it.
// We add a command "throw code result" to do it.
func cmdThrow(fr *Frame, argv []T) T {
	statusT, resultT := Arg2(argv)
	status := statusT.Int()

	panic(Jump{
		Status: StatusCode(status),
		Result: resultT,
	})
}

var stringEnsemble = []EnsembleItem{
	EnsembleItem{Name: "length", Cmd: cmdSLen},
	EnsembleItem{Name: "range", Cmd: cmdStringRange},
	EnsembleItem{Name: "slice", Cmd: cmdStringSlice},
	EnsembleItem{Name: "first", Cmd: cmdStringFirst},
	EnsembleItem{Name: "index", Cmd: cmdStringIndex},
	EnsembleItem{Name: "match", Cmd: cmdStringMatch},
	EnsembleItem{Name: "trim", Cmd: cmdStringTrim},
}

// Follows Tcl's string range spec.
func cmdStringRange(fr *Frame, argv []T) T {
	str, first, last := Arg3(argv)

	strS := str.String()
	n := len(strS)
	firstI := int(first.Int()) // The index of the first character to include.

	keep := 1     // Tcl's string range includes the character indexed by last
	var lastI int // The index of the last character to include.
	if last.IsEmpty() || last.String() == "end" {
		lastI = n - keep
	} else {
		lastI = int(last.Int())
	}

	low, high, ok := slicer(n, firstI, lastI, keep)
	if !ok {
		return Empty
	}

	return MkString(strS[low:high])
}

// Follows golang's slice spec.
func cmdStringSlice(fr *Frame, argv []T) T {
	str, first, last := Arg3(argv)

	strS := str.String()
	n := len(strS)
	firstI := int(first.Int()) // The index of the first character to include.

	var lastI int // The number characters to include.
	if last.IsEmpty() || last.String() == "end" {
		lastI = n
	} else {
		lastI = int(last.Int())
	}

	low, high, ok := slicer(n, firstI, lastI, 0)
	if !ok {
		return Empty
	}

	return MkString(strS[low:high])
}

// Slicer will find the low and high values for slicing a golang slice.
// http://golang.org/ref/spec#Slices
//
// Parameters:
// length - The length of the slice.
// first  - The index of the first element to take.
// last   - The high value for the slice.
//
//	If keep is 0, this will return a low/high value that will satisfy
//	0 <= low <= high <= length, like in go.
//
// keep   - The number of elements to keep.
//
// Returns:
// low    - The low value for the slice.
// high   - The high value for the slice.
// ok     - false if there is an invalid request.
func slicer(length int, first, last int, keep int) (int, int, bool) {
	// If first is too small, Zero.
	if first < 0 {
		first = 0
	}

	// If first is too large, Empty.
	if first > length {
		return -1, -1, false
	}

	// Last may be negative, like in Python.
	if last < 0 {
		last += length - keep
	}

	// If last is too small, Empty.
	if last < first {
		return -1, -1, false
	}

	// If last is too large, End.
	if last > length-keep {
		last = length - keep
	}

	return first, last + keep, true
}

// TODO: Add optional argument "startIndex"
func cmdStringFirst(fr *Frame, argv []T) T {
	needle, haystack := Arg2(argv)

	i := strings.Index(haystack.String(), needle.String())

	return MkFloat(float64(i))
}

func cmdStringIndex(fr *Frame, argv []T) T {
	str, charIndex := Arg2(argv)

	s := str.String()
	i := int(charIndex.Int())
	n := len(s)

	if i < 0 || i >= n {
		return Empty
	}

	z := string(s[i])
	return MkString(z)
}

func cmdStringTrim(fr *Frame, argv []T) T {
	t := Arg1(argv)
	s := t.String()
	for len(s) > 0 {
		if White(s[0]) {
			s = s[1:]
		} else {
			break
		}
	}
	for len(s) > 0 {
		if White(s[len(s)-1]) {
			s = s[:len(s)-1]
		} else {
			break
		}
	}
	return MkString(s)
}

func cmdStringMatch(fr *Frame, argv []T) T {
	pattern, str := Arg2(argv)

	return MkBool(StringMatch(pattern.String(), str.String()))
}

func StringMatch(pattern, str string) bool {
	plen, slen := len(pattern), len(str)
	pidx, cidx := 0, 0
	var p, c uint8

Loop:
	for pidx < plen {
		p = pattern[pidx]

		// c is unset.
		if p == '*' {
			// Skip successive *'s in the pattern
			for p == '*' {
				pidx++
				if pidx < plen {
					p = pattern[pidx]
				} else {
					return true
				}
			}

			// Loop through the string until satisfied.
			// p is the pattern after the * we found.
			// pidx != plen
			for cidx < slen {
				// Optimization:
				// If 'p' isn't a special character, look ahead for the next matching
				// character in the string.
				if p != '[' && p != '?' && p != '\\' {

					// c is the next character to try and match.
				StarLookAhead:
					for cidx < slen {
						c = str[cidx]

						if c == p {
							break StarLookAhead
						}

						cidx++

						// We reached the end of str so we can return early.
						if cidx == slen {
							return false
						}
					}
					// c should now be the first character that matches p
					// cidx should be the index of c in str
				}

				if StringMatch(pattern[pidx:], str[cidx:]) {
					return true
				}

				cidx++
			}
			// reached end of str
			// p is unmatched
			return false
		}

		if p == '?' {
			pidx++
			cidx++
			continue Loop
		}

		// Populate c if we can.
		if cidx < slen {
			c = str[cidx]
		} else {
			// We've run out of string.
			return false
		}

		if p == '[' {
			var start, end uint8

			// Skip the pidx to point to the next char
			pidx++

		BracketLoop:
			for {
				if pidx == plen {
					return false
				}

				p = pattern[pidx]
				if p == ']' {
					return false
				}

				start = p

				pidx++
				p = pattern[pidx]

				if p == '-' {
					// Match a range of characters.
					pidx++
					if pidx == plen {
						return false
					}

					p = pattern[pidx]
					end = p

					if (start <= c && c <= end) || (end <= c && c <= start) {
						break BracketLoop
					}
				} else if start == c {
					break BracketLoop
				}
			}

			// Skip to after the ending bracket.
			for p != ']' {
				pidx++
				if pidx < plen {
					p = pattern[pidx]
				} else {
					p--
					break
				}
			}

			// We succeeded in matching our character.  Continue the loop.
			pidx++
			cidx++
			continue Loop
		}

		// Strip off the '\' so we do an exact match on the following char.
		if p == '\\' {
			pidx++
			if pidx == plen {
				return false
			}

			p = pattern[pidx]
		}

		// The normal case, with no special characters.
		if c != p {
			return false
		}

		pidx++
		cidx++
	}

	// Are we at the end of both the pattern and the string?
	if pidx == plen {
		return cidx == slen
	}

	return false
}

var arrayEnsemble = []EnsembleItem{
	EnsembleItem{Name: "get", Cmd: cmdArrayGet},
	EnsembleItem{Name: "set", Cmd: cmdArraySet},
	EnsembleItem{Name: "size", Cmd: cmdArraySize},
	EnsembleItem{Name: "exists", Cmd: cmdArrayExists},
	EnsembleItem{Name: "names", Cmd: cmdArrayNames},
}

func cmdArraySet(fr *Frame, argv []T) T {
	varName, stuff := Arg2(argv)
	s := varName.String()
	v := stuff.List()
	n := len(v)
	if n%2 != 0 {
		panic("array set: got odd length of value list")
	}
	var h T
	if !fr.HasVar(s) {
		h = MkHash(nil)
		fr.SetVar(s, h)
	} else {
		h = fr.GetVar(s) // TODO: race
	}
	for i := 0; i < n; i += 2 {
		h.PutAt(v[i+1], v[i])
	}
	return h
}

func cmdArrayGet(fr *Frame, argv []T) T {
	varName := Arg1(argv) // TODO: optional glob.
	t := fr.GetVar(varName.String())
	h := t.Hash()

	var z []T
	for _, k := range SortedKeysOfHash(h) {
		z = append(z, MkString(k))
		z = append(z, h[k])
	}
	return MkList(z)
}
func cmdArraySize(fr *Frame, argv []T) T {
	name := Arg1(argv)
	s := name.String()

	if !fr.HasVar(s) {
		return Zero // Normal Tcl returns 0 if var doesn't exist.
	}

	t := fr.GetVar(s)
	h := t.Hash()
	n := len(h)
	return MkInt(int64(n))
}

func cmdArrayExists(fr *Frame, argv []T) T {
	name := Arg1(argv)
	s := name.String()

	if !fr.HasVar(s) {
		return False
	}
	t := fr.GetVar(s)
	_, ok := t.(*terpHash)
	return MkBool(ok)
}

func cmdArrayNames(fr *Frame, argv []T) T {
	hashName := Arg1(argv)
	t := fr.GetVar(hashName.String())
	h := t.Hash()
	return MkList(hashKeys(h))
}

var infoEnsemble = []EnsembleItem{
	EnsembleItem{Name: "macros", Cmd: cmdInfoMacros},
	EnsembleItem{Name: "commands", Cmd: cmdInfoCommands},
	EnsembleItem{Name: "globals", Cmd: cmdInfoGlobals},
	EnsembleItem{Name: "locals", Cmd: cmdInfoLocals},
	EnsembleItem{Name: "exists", Cmd: cmdInfoExists},
}

func cmdInfoMacros(fr *Frame, argv []T) T {
	Arg0(argv) // TODO: optional pattern
	var zz []T
	for k, _ := range fr.G.Macros {
		zz = append(zz, MkString(k))
	}
	SortListByString(zz)
	return MkList(zz)
}
func cmdInfoCommands(fr *Frame, argv []T) T {
	Arg0(argv) // TODO: optional pattern
	var zz []T
	for k, _ := range fr.G.Cmds {
		zz = append(zz, MkString(k))
	}
	SortListByString(zz)
	return MkList(zz)
}
func cmdInfoGlobals(fr *Frame, argv []T) T {
	Arg0(argv) // TODO: optional pattern
	var zz []T
	for k, _ := range fr.G.Fr.Vars {
		zz = append(zz, MkString(k))
	}
	SortListByString(zz)
	return MkList(zz)
}
func cmdInfoLocals(fr *Frame, argv []T) T {
	Arg0(argv) // TODO: optional pattern
	var zz []T
	for k, _ := range fr.Vars {
		zz = append(zz, MkString(k))
	}
	SortListByString(zz)
	return MkList(zz)
}

func cmdInfoExists(fr *Frame, argv []T) T {
	name := Arg1(argv)
	s := name.String()
	if strings.HasSuffix(s, ")") {
		p := strings.IndexByte(s, '(')
		if p < 0 {
			panic("bad syntax for array variable")
		}
		varname := s[:p]

		if !fr.HasVar(varname) {
			return False
		}
		t := fr.GetVar(varname)
		h, ok := t.(*terpHash)
		if !ok {
			return False
		}
		key := s[p+1 : len(s)-1]

		_, ok = h.h[key]
		return MkBool(ok)
	}

	return MkBool(fr.HasVar(s))
}

func cmdSplit(fr *Frame, argv []T) T {
	str, delimV := Arg1v(argv)
	s := str.String()
	if s == "" {
		return Empty // Special case in Tcl.
	}

	var delim string
	switch len(delimV) {
	case 0:
		delim = ""
	case 1:
		delim = delimV[0].String()
	default:
		panic("Usage: split str ?delims?")
	}
	if delim == "" {
		delim = " \t\n\r" // White Space.
	}

	z := make([]T, 0, 4)
	for {
		i := strings.IndexAny(s, delim)
		if i == -1 {
			z = append(z, MkString(s))
			break
		}
		z = append(z, MkString(s[:i]))
		s = s[i+1:]
	}
	return MkList(z)
}

func cmdJoin(fr *Frame, argv []T) T {
	list, joinV := Arg1v(argv)

	var joiner string
	switch len(joinV) {
	case 0:
		joiner = " "
	case 1:
		joiner = joinV[0].String()
	default:
		panic("Usage: join list ?joinString?")
	}

	buf := bytes.NewBuffer(nil)
	for i, e := range list.List() {
		if i > 0 {
			buf.WriteString(joiner)
		}
		buf.WriteString(e.String())
	}
	return MkString(buf.String())
}

func cmdSubst(fr *Frame, argv []T) T {
	args := Arg0v(argv)

	if len(args) == 0 {
		panic("'subst' commmand needs an argument")
	}

	var flags SubstFlags
	for len(args) > 1 {
		a := args[0].String()
		switch true {
		case StringMatch("-nob*", a):
			flags |= NoBackslash
		case StringMatch("-noc*", a):
			flags |= NoSquare
		case StringMatch("-nov*", a):
			flags |= NoDollar
		default:
			panic(Sprintf("Bad flag for 'subst' commmand: %q", a))
		}
		args = args[1:]
	}

	return MkString(fr.SubstString(args[0].String(), flags))
}

// Usage: log <level> <messages>...
// Creates a new stderr logger, if Global has no logger yet.
func cmdLog(fr *Frame, argv []T) T {
	levelT, messageT := Arg2(argv)
	Log(fr, levelT.String(), messageT.String())
	return Empty
}

func Log(fr *Frame, levelStr string, message string) {
	var panicky, fatally bool
	if len(levelStr) != 1 {
		panic(Sprintf("Log level should be 'p', 'f', or in '0'..'9' but is %q", levelStr))
	}
	lev := levelStr[0]
	level := -1 // for case 'p' or 'f'

	if lev == 'p' { // "p"anic level
		panicky = true
	} else if lev == 'f' { // "f"atal level
		fatally = true
	} else if '0' <= lev && lev <= '9' {
		level = int(lev) - int('0')
	} else {
		panic(Sprintf("Log level should be 'p', 'f', or in '0'..'9' but is %q", level))
	}

	/*
		if level > fr.G.Verbosity {
			return // Not enough verbosity for this message.
		}
	*/

	if fr.G.Logger == nil {
		logName := fr.G.LogName
		if logName == "" {
			logName = "chirp" // Default LogName
		}
		fr.G.Logger = log.New(os.Stderr, logName, log.LstdFlags)
	}

	message = SubstStringOrOrig(fr, message)

	fr.G.Logger.Println(message)

	if panicky {
		panic(Sprintf("log p: %s", message))
	}
	if fatally {
		fr.G.Logger.Println("Exiting after fatal log message.")
		os.Exit(13) // Unlucky Exit.
	}
}

func SubstStringOrOrig(fr *Frame, s string) (z string) {
	defer func() {
		if r := recover(); r != nil {
			z = Sprintf("ERROR ignored while substituting log message: %s", s)
			return
		}
	}()
	return fr.SubstString((s), 0) // 0 is all substitutions.
}

func init() {
	if Safes == nil {
		Safes = make(map[string]Command, 333)
	}

	Safes["must"] = cmdMust
	Safes["mustfail"] = cmdMustFail
	Safes["if"] = cmdIf
	Safes["case"] = cmdCase
	Safes["format"] = cmdFormat
	Safes["scan"] = cmdScan
	Safes["echo"] = cmdEcho
	Safes["say"] = cmdSay
	Safes["macro"] = cmdMacro
	Safes["proc"] = cmdProc

	Safes["list"] = cmdList
	Safes["lindex"] = cmdLIndex
	Safes["lrange"] = cmdLRange
	Safes["lsort"] = cmdLSort
	Safes["lreverse"] = cmdLReverse
	Safes["llength"] = cmdLLen
	Safes["foreach"] = cmdForEach
	Safes["while"] = cmdWhile
	Safes["catch"] = cmdCatch
	Safes["eval"] = cmdEval
	Safes["go"] = cmdGo
	Safes["uplevel"] = cmdUpLevel
	Safes["concat"] = cmdConcat
	Safes["set"] = cmdSet
	Safes["global"] = cmdGlobal
	Safes["upvar"] = cmdUpVar
	Safes["return"] = cmdReturn
	Safes["break"] = cmdBreak
	Safes["continue"] = cmdContinue

	// Keep these even though they are quirky.
	Safes["hash"] = cmdHash
	Safes["hget"] = cmdHGet
	Safes["hset"] = cmdHSet
	Safes["hdel"] = cmdHDel
	Safes["hkeys"] = cmdHKeys

	Safes["incr"] = cmdIncr
	Safes["append"] = cmdAppend
	Safes["lappend"] = cmdLAppend
	Safes["error"] = cmdError
	Safes["throw"] = cmdThrow
	Safes["string"] = MkEnsemble(stringEnsemble)
	Safes["info"] = MkEnsemble(infoEnsemble)
	Safes["array"] = MkEnsemble(arrayEnsemble)
	Safes["split"] = cmdSplit
	Safes["join"] = cmdJoin
	Safes["subst"] = cmdSubst
	Safes["log"] = cmdLog
	Safes["usage"] = cmdUsage // TODO?
	Safes["time"] = cmdTime
	Safes["clock"] = MkEnsemble(clockEnsemble)
}
