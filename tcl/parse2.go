/*
	parse2 will parse to an ADT and support Visitor Pattern on it.
	This will be useful for bytecodes, compilers, tools.
*/

package tcl

import (
	"bytes"
	. "fmt"
	"regexp"
	_ "strings"
)

// An expr command
type PExpr struct {
	Op      Token
	A, B, C *PExpr
	Word    *PWord // Op '"': for Primatives: "quoted", [square], 3.14 literal, {literal}, $var, $var(index).
}

func (me *PExpr) Eval(fr *Frame) T {
	Parse2ExprEvalCounter.Incr()
	switch me.Op {
	case '+':
		a := me.A.Eval(fr)
		b := me.B.Eval(fr)
		if a.IsQuickInt() && b.IsQuickInt() {
			return MkInt(a.Int() + b.Int())
		}
		return MkFloat(a.Float() + b.Float())
	case '-':
		if me.B == nil {
			// Unary minus, if me.B is missing.
			a := me.A.Eval(fr)
			if a.IsQuickInt() {
				return MkInt(0 - a.Int())
			}
			return MkFloat(0.0 - a.Float())
		}
		a := me.A.Eval(fr)
		b := me.B.Eval(fr)
		if a.IsQuickInt() && b.IsQuickInt() {
			return MkInt(a.Int() - b.Int())
		}
		return MkFloat(a.Float() - b.Float())
		return MkFloat(me.A.Eval(fr).Float() - me.B.Eval(fr).Float())
	case '*':
		a := me.A.Eval(fr)
		b := me.B.Eval(fr)
		if a.IsQuickInt() && b.IsQuickInt() {
			return MkInt(a.Int() * b.Int())
		}
		return MkFloat(a.Float() * b.Float())
	case '/':
		a := me.A.Eval(fr)
		b := me.B.Eval(fr)
		Say("a/b", a, b)
		if a.IsQuickInt() && b.IsQuickInt() {
			Say("MkInt", a, b)
			return MkInt(a.Int() / b.Int())
		}
		Say("MkFloat", a, b)
		return MkFloat(a.Float() / b.Float())
	case '%':
		return MkInt(me.A.Eval(fr).Int() % me.B.Eval(fr).Int())
	case '&':
		return MkUint(uint64(BitsWord(me.A.Eval(fr).Uint()) % BitsWord(me.B.Eval(fr).Uint())))
	case '|':
		return MkUint(uint64(BitsWord(me.A.Eval(fr).Uint()) | BitsWord(me.B.Eval(fr).Uint())))
	case '^':
		return MkUint(uint64(BitsWord(me.A.Eval(fr).Uint()) ^ BitsWord(me.B.Eval(fr).Uint())))
	case TokBoolAnd:
		return MkBool(me.A.Eval(fr).Bool() && me.B.Eval(fr).Bool())
	case TokBoolOr:
		return MkBool(me.A.Eval(fr).Bool() || me.B.Eval(fr).Bool())
	case '!':
		return MkBool(!me.A.Eval(fr).Bool())
	case '~':
		return MkUint(uint64(^BitsWord(me.A.Eval(fr).Uint())))
	case '"': // For "quoted" and {curlied} and $Var and $Var(index)
		return me.Word.Eval(fr)
	case '?':
		if me.A.Eval(fr).Bool() {
			return me.B.Eval(fr)
		} else {
			return me.C.Eval(fr)
		}
	case '<':
		return MkBool(me.A.Eval(fr).Float() < me.B.Eval(fr).Float())
	case '>':
		return MkBool(me.A.Eval(fr).Float() > me.B.Eval(fr).Float())
	case TokNumEq:
		return MkBool(me.A.Eval(fr).Float() == me.B.Eval(fr).Float())
	case TokNumNe:
		return MkBool(me.A.Eval(fr).Float() != me.B.Eval(fr).Float())
	case TokNumLe:
		return MkBool(me.A.Eval(fr).Float() <= me.B.Eval(fr).Float())
	case TokNumGe:
		return MkBool(me.A.Eval(fr).Float() >= me.B.Eval(fr).Float())
	case TokStrLt:
		return MkBool(me.A.Eval(fr).String() < me.B.Eval(fr).String())
	case TokStrGt:
		return MkBool(me.A.Eval(fr).String() > me.B.Eval(fr).String())
	case TokStrEq:
		return MkBool(me.A.Eval(fr).String() == me.B.Eval(fr).String())
	case TokStrNe:
		return MkBool(me.A.Eval(fr).String() != me.B.Eval(fr).String())
	case TokStrLe:
		return MkBool(me.A.Eval(fr).String() <= me.B.Eval(fr).String())
	case TokStrGe:
		return MkBool(me.A.Eval(fr).String() >= me.B.Eval(fr).String())
	}
	panic(Sprintf("PANIC PExpr.Eval unknown op: %d", me.Op))
}

func (me *PExpr) Show() string {
	z := "PExpr{ "
	if me.Op < 33 {
		z += Sprintf("op%d ", me.Op)
	} else {
		z += Sprintf("op%c ", me.Op)
	}
	if me.A != nil {
		z += me.A.Show()
	}
	if me.B != nil {
		z += me.B.Show()
	}
	if me.C != nil {
		z += me.C.Show()
	}
	if me.Word != nil {
		z += me.Word.Show()
	}
	z += "} "
	return z
}

// Any piece of tcl code, a sequence of commands.
type PSeq struct {
	Cmds []*PCmd
	Src  string
}

func (me *PSeq) Eval(fr *Frame) T {
	Parse2SeqEvalCounter.Incr()
	var z T = Empty
	for _, cmd := range me.Cmds {
		z = cmd.Eval(fr)
	}
	return z
}

func (me *PSeq) Show() string {
	z := "PSeq{ "
	for _, e := range me.Cmds {
		z += e.Show()
	}
	z += "} "
	return z
}

// One command made of one or more words.
type PCmd struct {
	Words []*PWord
}

func (me *PCmd) Eval(fr *Frame) T {
	if Debug['w'] {
		Say("PCmd.Eval: ", me.Show())
	}
	Parse2CmdEvalCounter.Incr()

	var words []T
	for _, w := range me.Words {
		if w.ExpandAsMultiWord {
			t := w.Eval(fr)
			for _, e := range t.List() {
				words = append(words, e)
			}
		} else {
			words = append(words, w.Eval(fr))
		}
	}

	// Send Apply to the first word.
	z := words[0].Apply(fr, words)
	if Debug['w'] {
		Say("PCmd.Eval: Return: ", z)
	}
	return z
}

func (me *PCmd) Show() string {
	z := "PCmd{ "
	for _, e := range me.Words {
		z += e.Show()
	}
	z += "} "
	return z
}

// One words, composed of parts that may require substitions.
type PWord struct {
	Parts             []*PPart
	Multi             *terpMulti // If not null, value is fixed and precompiled.
	ExpandAsMultiWord bool
}

func (me *PWord) Eval(fr *Frame) (z T) {
	if Debug['w'] {
		Say("PWord.Eval: ", me.Show())
	}
	switch len(me.Parts) {
	case 0:
		Parse2WordEvalFastCounter0.Incr()
		z = Empty
	case 1:
		if me.Multi != nil {
			Parse2WordEvalFastCounter1.Incr()
			z = me.Multi
		} else {
			Parse2WordEvalSlowCounter1.Incr()
			z = me.Parts[0].Eval(fr)
		}
	default:
		Parse2WordEvalSlowCounter9.Incr()
		buf := bytes.NewBuffer(nil)
		for _, part := range me.Parts {
			if part.Type == BARE { // Optimization: avoid creating a T.
				buf.WriteString(part.Multi.s.s)
			} else {
				buf.WriteString(part.Eval(fr).String())
			}
		}
		z = MkString(buf.String())
	}
	if Debug['w'] {
		Say("PWord.Eval: Returns:", me.Show())
	}
	return z
}

func (me *PWord) Show() string {
	z := "PWord{ "
	if me.ExpandAsMultiWord {
		z += "(EXPAND) "
	}
	for _, e := range me.Parts {
		z += e.Show()
	}
	z += "} "
	return z
}

type PartType int

const (
	BARE    PartType = iota + 1 // Does not need substitions (backslash subs aready done).
	DOLLAR1                     // $x, variable subs without index
	DOLLAR2                     // $x(...), variable subs with index
	SQUARE                      // [...], subcommand eval and replace.
)

type PPart struct {
	VarName string
	Multi   *terpMulti
	Word    *PWord // for DOLLAR2
	Seq     *PSeq  // for SQUARE
	Type    PartType
}

func (me *PPart) Eval(fr *Frame) T {
	switch me.Type {
	case BARE:
		return me.Multi
	case SQUARE:
		return me.Seq.Eval(fr)
	case DOLLAR1:
		v := fr.GetVar(me.VarName)
		if v == nil {
			panic(Sprintf("(* PWord.Eval.DOLLAR1 *) Variable %q does not exist.", me.VarName))
		}
		return v
	case DOLLAR2:
		v := fr.GetVar(me.VarName)
		if v == nil {
			panic(Sprintf("(* PWord.Eval.DOLLAR2 *) Variable %q does not exist.", me.VarName))
		}
		h := v.Hash()
		if h == nil {
			panic(Sprintf("(* PWord.Eval.DOLLAR2 *) Variable %q is not a hash.", me.VarName))
		}

		k := me.Word.Eval(fr).String()
		z, ok := h[k]
		if !ok {
			panic(Sprintf("(*PWord.Eval.DOLLAR2*) Variable %q: Key not found", me.VarName))
		}
		return z
	}
	panic(Sprintf("(*PWord.Eval*) Unknown PartType: %d", me.Type))
}

func (me *PPart) Show() string {
	switch me.Type {
	case BARE:
		return Sprintf("BARE{%#v} ", me.Multi.Show())
	case DOLLAR1:
		return Sprintf("DOLLAR1{%#v} ", me.VarName)
	case DOLLAR2:
		return Sprintf("DOLLAR2{%#v,%s} ", me.VarName, me.Word.Show())
	case SQUARE:
		return Sprintf("SQUARE{ %s } ", me.Seq.Show())
	}
	return Sprintf("PPart{?%d?} ", me.Type)
}

// Parse nested curlies, returning contents.
func Parse2Curly(lex *Lex) *PWord {
	if lex.Tok != '{' {
		panic("Parse2Curly should begin at open curly")
	} // vim: '}'

	x := lex.AdvanceCurly()
	// Next is now on the close-curly.
	// Let set Tok to the close-curly:
	lex.Advance()

	multi := MkMulti(x)
	result := &PWord{
		Parts: []*PPart{
			&PPart{
				Multi: multi,
				Type:  BARE,
			},
		},
		Multi: multi,
	}
	return result
}

// finishBarePart turns the buffer into a BARE PPart and appends it to the slice, unless the buffer is empty.
// It returns a new empty buffer (or the input buffer, if it was empty).
// This pattern is used by most of the Parsers.
func finishBarePart(parts []*PPart, buf *bytes.Buffer) ([]*PPart, *bytes.Buffer) {
	if buf.Len() > 0 {
		bare := &PPart{
			Type:  BARE,
			Multi: MkMulti(buf.String()),
		}
		parts = append(parts, bare)
		if Debug['p'] {
			Say("finishBarePart ->", buf.String(), parts)
		}
		return parts, bytes.NewBuffer(nil)
	}
	if Debug['p'] {
		Say("finishBarePart -> empty ->", parts)
	}
	return parts, buf
}

// Parse Square Bracketed subcommand, returning result and new position
func Parse2Square(lex *Lex) *PPart {
	Parse2SquareCounter.Incr()
	if lex.Tok != Token('[') && lex.Tok != TokExpandSquare {
		panic("Parse2Square should begin at open square")
	}
	lex.Advance()
	begin := lex.Pos
	cmds := make([]*PCmd, 0)

Loop:
	for {
		var cmd *PCmd
		cmd = Parse2Cmd(lex)
		if cmd == nil {
			break Loop
		}
		cmds = append(cmds, cmd)
	}

	if lex.Tok != Token(']') {
		panic(Sprintf("Parse2Square: missing end bracket: rest=%q" + lex.Str[lex.Next:]))
	}
	end := lex.Pos
	return &PPart{Type: SQUARE, Seq: &PSeq{Cmds: cmds, Src: lex.Str[begin:end]}}
}

func Parse2Quote(lex *Lex) *PWord {
	Parse2QuoteCounter.Incr()
	if lex.Tok != Token('"') {
		panic("PANIC: Parse2Quote should begin at open Quote")
	}
	buf := bytes.NewBuffer(nil)
	parts := make([]*PPart, 0)

Loop:
	for lex.Next < lex.Len {
		c := lex.Str[lex.Next]
		switch c {
		case '"':
			// lex.Stretch1()
			lex.Advance() // focus on close-quote.
			break Loop
		case '[':
			parts, buf = finishBarePart(parts, buf)
			// Mid-word, squares should return stringlike result.
			lex.Advance() // to Token('[')
			MustTok(Token('['), lex.Tok)
			r := Parse2Square(lex)
			MustTok(Token(']'), lex.Tok)
			parts = append(parts, r)
		case ']':
			panic("Parse2Quote: CloseSquareBracket inside Quote")
		case '$':
			parts, buf = finishBarePart(parts, buf)
			lex.Advance() // to Token('$')
			MustTok(Token('$'), lex.Tok)
			r := Parse2Dollar(lex)
			parts = append(parts, r)
		case '\\':
			c = lex.StretchBackslashEscaped()
			buf.WriteByte(c)
		default:
			buf.WriteByte(c)
			lex.Stretch1()
		}
	}
	parts, buf = finishBarePart(parts, buf)
	return &PWord{Parts: parts}
}

// Parse a word, returning result and new position
func Parse2Word(lex *Lex) *PWord {
	buf := bytes.NewBuffer(nil)
	parts := make([]*PPart, 0)

Loop:
	for lex.Tok != TokEnd {
		if Debug['p'] {
			Say("Word: LOOP: buf", buf.String())
			Say("Word: LOOP: Tok=", lex.Tok)
			Say("Word: LOOP: show=", lex)
		}

		// Use the normal Tok lexer to start,
		// but finish with Next pointing to the next unconsumed thing,
		// either more parts to the word, or white space.
		switch lex.Tok {
		case '[':
			parts, buf = finishBarePart(parts, buf)
			// Mid-word, squares should return stringlike result.
			r := Parse2Square(lex)
			parts = append(parts, r)
			MustTok(Token(']'), lex.Tok)

		case ']':
			// The close-bracket is not part of the word;
			// it terminates an outer seq of cmds.
			// Break with focus still on the bracket.
			break Loop

		case '$':
			parts, buf = finishBarePart(parts, buf)
			r := Parse2Dollar(lex)
			parts = append(parts, r)
			// Next should be just after the whole DOLLAR thing.

		case TokNewline:
			parts, buf = finishBarePart(parts, buf)
			// lex.Advance() // MAYBE
			break Loop

		case '"':
			panic("Parse2Word: DoubleQuote inside word")

		case '\\':
			lex.Next = lex.Pos // StretchBackslashEscaped wants Next to point to the backslash.
			c := lex.StretchBackslashEscaped()
			buf.WriteByte(c)

		default:
			if Debug['p'] {
				Say("Word: Loop: DEFAULT: Tok=", lex.Tok)
				Say("Word: Loop: DEFAULT: Show=", lex.Show())
				Say("Word: Loop: DEFAULT: Current=", lex.Current())
			}
			buf.WriteString(lex.Current())
		}

		// Peek to see if white space is next.
		if lex.Next < lex.Len {
			switch lex.Str[lex.Next] {
			case ' ', '\t', '\r', '\v', '\n', ';':
				// If white space, then this Word is over.  Break.
				parts, buf = finishBarePart(parts, buf)
				lex.Advance() // Advance past this white space (but not past \n or ;) to next thing before break.
				break Loop
			}
		}

		// Not at white space; there is more.
		// Pos will become Next.  Already made sure not at white space.
		// This will focus on the next thing in the Word.
		lex.Advance()
	}

	parts, buf = finishBarePart(parts, buf)
	z := &PWord{Parts: parts}
	if len(parts) == 1 && parts[0].Type == BARE {
		z.Multi = parts[0].Multi // Optimize for fixed bare value.
	}
	if Debug['p'] {
		Say("Word: z ->", z)
	}
	return z
}

// Parse the Key for a Dollar with Parens, e.g. $x(key).
// Dollar, Square, and Backslash substitutions occur.
// White space and DQuotes are not special.
// Terminates with ")".
func Parse2DollarKey(lex *Lex) *PWord {
	buf := bytes.NewBuffer(nil)
	parts := make([]*PPart, 0)

	MustB('(', lex.Str[lex.Next])
	lex.Stretch1()

Loop:
	for lex.Next < lex.Len {
		c := lex.PeekNext()
		switch c {
		case ')':
			break Loop
		case '[':
			parts, buf = finishBarePart(parts, buf)
			// Mid-word, squares should return stringlike result.
			lex.Advance()
			MustTok(Token('['), lex.Tok)
			r := Parse2Square(lex)
			parts = append(parts, r)
			MustTok(Token(']'), lex.Tok)
		case '$':
			parts, buf = finishBarePart(parts, buf)
			lex.Advance()
			MustTok(Token('$'), lex.Tok)
			r := Parse2Dollar(lex)
			parts = append(parts, r)
		case '\\':
			c = lex.StretchBackslashEscaped()
			buf.WriteByte(c)
		default:
			buf.WriteByte(c)
			lex.Stretch1()
		}
	}
	lex.Advance()
	MustTok(Token(')'), lex.Tok)

	parts, buf = finishBarePart(parts, buf)
	return &PWord{Parts: parts}
}

// Parse a variable name after a '$', returning *PPart.  Leaves Next immediately after the consumed part.
func Parse2Dollar(lex *Lex) *PPart {
	Parse2DollarCounter.Incr()
	if lex.Tok != Token('$') && lex.Tok != TokExpandDollar {
		panic("Expected $ at beginning of Parse2Dollar")
	}

	lex.AdvanceIfAlfaNum()
	switch lex.Tok {
	case TokAlfaNum, TokStrEq, TokStrNe, TokStrLt, TokStrLe, TokStrGt, TokStrGe:

	default:
		panic("Expected a varname after $")
	}
	name := lex.Current()

	if lex.PeekNext() == '(' {
		key := Parse2DollarKey(lex)
		MustTok(Token(')'), lex.Tok)
		// DONT lex.Stretch1()
		return &PPart{
			Type:    DOLLAR2,
			VarName: name,
			Word:    key,
		}
	}
	// else:
	return &PPart{
		Type:    DOLLAR1,
		VarName: name,
	}
}

func FollowedByGap(lex *Lex) bool {
	nc := byte(';')
	if len(lex.Str) > lex.Next {
		nc = byte(lex.Str[lex.Next])
	}
	return nc == '\n' || nc == ';' || nc == ']' || nc == ' ' || nc == '\t' || nc == '\r' || nc == '\v'
}

// Returns next command, or else nil.
func Parse2Cmd(lex *Lex) *PCmd {
	Parse2CmdCounter.Incr()
	var words []*PWord
	if Debug['p'] {
		Say("Parse2Cmd <<<", lex)
	}

Restart:
	// skip initial newlines and ';'s (as well as white space)
	for lex.Tok == Token(';') || lex.Tok == TokNewline {
		lex.Advance()
	}

Loop:
	// Ways break Loop: TokEnd, TokNewline, Token(';'), Token(']').
	// We leave the lex.Tok at one of those four when we return.
	for lex.Tok != TokEnd {
		switch lex.Tok {
		case TokNewline, Token(';'):
			lex.Advance()
			break Loop
		case Token(']'):
			// DONT lex.Advance() -- leave this at the end-bracket.
			break Loop
		case Token('{'): // vim: '}'
			// TODO: Look for {*} prefix.
			r := Parse2Curly(lex)
			words = append(words, r)
			// vim: '{'
			MustTok(Token('}'), lex.Tok)

			if !FollowedByGap(lex) {
				panic(Sprintf("braces not followed by end of word"))
			}

			lex.Advance()
		case TokExpandSquare:
			part := Parse2Square(lex)
			words = append(words, &PWord{Parts: []*PPart{part}, ExpandAsMultiWord: true})
			MustTok(Token(']'), lex.Tok)
			if !FollowedByGap(lex) {
				panic(Sprintf("expanding brackets not followed by end of word"))
			}
			lex.Advance()
		case TokExpandDollar:
			part := Parse2Dollar(lex)
			words = append(words, &PWord{Parts: []*PPart{part}, ExpandAsMultiWord: true})
			if !FollowedByGap(lex) {
				panic(Sprintf("expanding brackets not followed by end of word"))
			}
			lex.Advance()
		case Token('['):
			part := Parse2Square(lex)
			// TODO: Bug if word is [foo][bar]
			words = append(words, &PWord{Parts: []*PPart{part}})
			// vim: '['
			MustTok(Token(']'), lex.Tok)
			// TODO: MIGHT NOT Be Followed By White Or End
			lex.Advance()
		case Token('"'):
			r := Parse2Quote(lex)
			words = append(words, r)
			MustTok(Token('"'), lex.Tok)
			lex.Advance()
		case Token('#'):
			if len(words) == 0 {
				lex.SkipComment()
				goto Restart
			}
			fallthrough // # is not special if not at beginning of command.
		default:
			r := Parse2Word(lex)
			words = append(words, r)
		}
	} // End Loop

	if len(words) == 0 {
		return nil
	}
	if Debug['p'] {
		Say("Parse2Cmd >>>", words)
		Say("Parse2Cmd >>>", lex)
	}
	return &PCmd{Words: words}
}

func Parse2Seq(lex *Lex) *PSeq {
	Parse2SeqCounter.Incr()
	begin := lex.Pos
	z := &PSeq{
		Cmds: make([]*PCmd, 0),
	}
Loop:
	for {
		cmd := Parse2Cmd(lex)
		if cmd == nil {
			break Loop
		}
		z.Cmds = append(z.Cmds, cmd)
	}
	end := lex.Pos
	z.Src = lex.Str[begin:end]
	return z
}

func Parse2ExprPrimative(lex *Lex) *PExpr {
	switch lex.Tok {
	case '"':
		word := Parse2Quote(lex)
		lex.Advance()
		return &PExpr{Op: '"', Word: word}
	case '[':
		part := Parse2Square(lex)
		lex.Advance()
		word := &PWord{Parts: []*PPart{part}}
		return &PExpr{Op: '"', Word: word}
	case '{':
		word := Parse2Curly(lex)
		lex.Advance()
		return &PExpr{Op: '"', Word: word}
	case TokNumber:
		// TODO: use MkMulti, and change Str to a T.
		part := &PPart{Multi: MkMulti(lex.Current()), Type: BARE}
		lex.Advance()
		word := &PWord{Parts: []*PPart{part}}
		return &PExpr{Op: '"', Word: word}
	case '$':
		part := Parse2Dollar(lex) // Leaves Next.
		lex.Advance()             // Advance to next Tok.
		word := &PWord{Parts: []*PPart{part}}
		return &PExpr{Op: '"', Word: word}
	case '(':
		lex.Advance()
		inner := Parse2ExprTop(lex)
		MustTok(')', lex.Tok)
		lex.Advance()
		return inner
	}
	panic(Sprintf("Expected Primative in Expr: %s", lex.Show()))
}

func Parse2ExprUnary(lex *Lex) *PExpr {
	switch lex.Tok {
	case '!', '~', '-':
		t := lex.Tok
		lex.Advance()
		b := Parse2ExprUnary(lex)
		return &PExpr{Op: t, A: b}
	}
	return Parse2ExprPrimative(lex)
}

func Parse2ExprProduct(lex *Lex) *PExpr {
	z := Parse2ExprUnary(lex)
	switch lex.Tok {
	case '*', '/', '%', '&', TokShiftLeft, TokShiftRight:
		t := lex.Tok
		lex.Advance()
		b := Parse2ExprProduct(lex)
		z = &PExpr{Op: t, A: z, B: b}
	}
	return z
}

func Parse2ExprSum(lex *Lex) *PExpr {
	z := Parse2ExprProduct(lex)
	switch lex.Tok {
	case '+', '-', '|', '^':
		t := lex.Tok
		lex.Advance()
		b := Parse2ExprSum(lex)
		z = &PExpr{Op: t, A: z, B: b}
	}
	return z
}

func Parse2ExprRelation(lex *Lex) *PExpr {
	z := Parse2ExprSum(lex)
	switch lex.Tok {
	case TokNumEq, TokNumNe, '<', TokNumLe, '>', TokNumGe,
		TokStrEq, TokStrNe, TokStrLt, TokStrLe, TokStrGt, TokStrGe:
		t := lex.Tok
		lex.Advance()
		b := Parse2ExprRelation(lex)
		z = &PExpr{Op: t, A: z, B: b}
	}
	return z
}

func Parse2ExprConjunction(lex *Lex) *PExpr {
	z := Parse2ExprRelation(lex)
	if lex.Tok == TokBoolAnd {
		lex.Advance()
		b := Parse2ExprConjunction(lex)
		z = &PExpr{Op: TokBoolAnd, A: z, B: b}
	}
	return z
}

func Parse2ExprDisjunction(lex *Lex) *PExpr {
	z := Parse2ExprConjunction(lex)
	if lex.Tok == TokBoolOr {
		lex.Advance()
		b := Parse2ExprDisjunction(lex)
		z = &PExpr{Op: TokBoolOr, A: z, B: b}
	}
	return z
}

func Parse2ExprTop(lex *Lex) *PExpr {
	Parse2ExprTopCounter.Incr()
	z := Parse2ExprDisjunction(lex)
	if Debug['x'] {
		Say("Top -> z", z, lex, lex.Tok, string(lex.Tok))
	}

	if lex.Tok == '?' {
		lex.Advance()
		b := Parse2ExprTop(lex)
		if Debug['x'] {
			Say("Top -> -> z, b", z, b, lex, lex.Tok, string(lex.Tok))
		}
		MustTok(Token(':'), lex.Tok)
		lex.Advance()
		c := Parse2ExprTop(lex)
		z = &PExpr{Op: '?', A: z, B: b, C: c}
		if Debug['x'] {
			Say("Top -> -> -> c, z", c, z, lex, lex.Tok, string(lex.Tok))
		}
	}
	// NOT// We must finish either at the End or at '}'
	// if lex.Tok != TokEnd && lex.Tok != '}' && lex.Tok != ')' {
	//	panic(Sprintf("Extra stuff after expression: %s", lex.Show()))
	// }
	return z
}

func Parse2ExprStr(s string) *PExpr {
	lex := NewLex(s)
	z := Parse2ExprTop(lex)
	MustTok(TokEnd, lex.Tok)
	return z
}

func Parse2SeqStr(s string) *PSeq {
	if Debug['p'] {
		Say("Parse2SeqStr <<<", s)
	}
	lex := NewLex(s)
	seq := Parse2Seq(lex)
	MustTok(TokEnd, lex.Tok)
	if Debug['p'] {
		Say("Parse2SeqStr >>>", seq)
	}
	return seq
}

func Parse2EvalSeqStr(fr *Frame, s string) T {
	if Debug['p'] {
		Say("Parse2EvalSeqStr <<<", s)
	}
	lex := NewLex(s)
	seq := Parse2Seq(lex)
	MustTok(TokEnd, lex.Tok)
	if Debug['p'] {
		Say("Parse2EvalSeqStr >>> seq=", seq)
	}
	z := seq.Eval(fr)
	if Debug['p'] {
		Say("Parse2EvalSeqStr >>> z=", z)
	}
	return z
}

func Parse2EvalExprStr(fr *Frame, s string) T {
	lex := NewLex(s)
	expr := Parse2ExprTop(lex)
	MustTok(TokEnd, lex.Tok)
	z := expr.Eval(fr)
	return z
}

//////////////////

var InertChars = "[!%-/0-9:<-@A-Z^_`a-z|~]"
var BareWordPattern = "(" + InertChars + "+)"

var AlphanumChars = "[A-Za-z0-9_]"
var DumbDollarPattern = "[$](" + AlphanumChars + "+)"

var MatchBareWord = regexp.MustCompile("^" + BareWordPattern + "$")
var MatchDumbDollar = regexp.MustCompile("^" + DumbDollarPattern + "$")

func CompileSequence(fr *Frame, s string) *PSeq {
	lex := NewLex(s)
	z := Parse2Seq(lex)
	if lex.Tok != TokEnd {
		Sayf("CompileSequence Non-Empty rest: %q", s)
		return nil
	}
	if Debug['p'] {
		Say("CompileSequence <<<<<<", z.Show())
	}
	maxSubCompile := len(s) // Stop infinite recursion sub-compiles.
	z2 := z.ExpandMacros(fr, maxSubCompile)
	if Debug['p'] {
		Say("CompileSequence <<<<<<", z.Show(), ">>>>>>", z2.Show())
	}
	return z2
}

func (me *PSeq) CloneAndSubst(params map[string][]*PWord) *PSeq {
	var zz []*PCmd
	for _, c := range me.Cmds {
		zz = append(zz, c.CloneAndSubst(params))
	}

	return &PSeq{
		Cmds: zz,
		Src:  me.Src,
	}
}
func (me *PSeq) ExpandMacros(fr *Frame, maxSubCompile int) *PSeq {
	var zz []*PCmd
	for _, c := range me.Cmds {
		zz = append(zz, c.ExpandMacros(fr, maxSubCompile)...)
	}

	return &PSeq{
		Cmds: zz,
		Src:  me.Src,
	}
}

func (me *PCmd) CloneAndSubst(params map[string][]*PWord) *PCmd {
	var zz []*PWord
	for _, word := range me.Words {
		zz = append(zz, word.CloneAndSubst(params)...)
	}
	return &PCmd{Words: zz}
}
func (me *PCmd) ExpandMacros(fr *Frame, maxSubCompile int) []*PCmd {
	if me.Words != nil {

		hd := me.Words[0]
		if hd.Multi != nil {
			name := hd.Multi.String()
			if macro, ok := fr.G.Macros[name]; ok {
				params := make(map[string][]*PWord)
				lenArgs := len(macro.Args)
				if lenArgs > 0 && macro.Args[lenArgs-1] == "ARGS" {
					// Final param ARGS is special, getting a list of the rest of actual params.
					if lenArgs-1 > len(me.Words)-1 {
						panic(Sprintf("Too few args to macro %q, got %d, wanted at least %d.", name, lenArgs-1, len(me.Words)-1))
					}

					// All but the last one.
					for i, p := range macro.Args[:lenArgs-1] {
						if p == "ARGS" {
							panic("Macro cannot have ARGS except as last parameter.")
						}
						params[p] = []*PWord{me.Words[i+1]}
					}

					// The last one.
					var vec []*PWord
					for _, w := range me.Words[lenArgs-0:] {
						vec = append(vec, w)
					}
					params["ARGS"] = vec
				} else {
					// Exact match needed between formal and actual params.
					if lenArgs != len(me.Words)-1 {
						panic(Sprintf("Wrong number of args to macro %q, got %d, wanted %d.", name, len(me.Words)-1, lenArgs))
					}

					for i, p := range macro.Args {
						if p == "ARGS" {
							panic("Macro cannot have ARGS except as last parameter.")
						}
						params[p] = []*PWord{me.Words[i+1]}
					}
				}

				var zz []*PCmd
				for _, clone := range macro.Body.CloneAndSubst(params).Cmds {
					zz = append(zz, clone.ExpandMacros(fr, maxSubCompile)...)
				}
				return zz
			}
		}
	}
	var zz []*PWord
	for _, word := range me.Words {
		zz = append(zz, word.ExpandMacros(fr, maxSubCompile))
	}
	return []*PCmd{&PCmd{Words: zz}}
}

func (me *PWord) CloneAndSubst(params map[string][]*PWord) []*PWord {
	if me.Multi != nil {
		return []*PWord{me}
	}

	if me.ExpandAsMultiWord && len(me.Parts) == 1 && me.Parts[0].Type == DOLLAR1 && me.Parts[0].VarName == "ARGS" {
		if vec, ok := params["ARGS"]; ok {
			return vec
		}
	}

	var zz []*PPart
	for _, part := range me.Parts {
		zz = append(zz, part.CloneAndSubst(params)...)
	}
	return []*PWord{&PWord{Parts: zz, ExpandAsMultiWord: me.ExpandAsMultiWord}}
}

func (me *PWord) ExpandMacros(fr *Frame, maxSubCompile int) *PWord {
	var zz []*PPart
	for _, part := range me.Parts {
		zz = append(zz, part.ExpandMacros(fr, maxSubCompile))
	}

	// Special case of 1 BARE part: Set the Multi field in the word.
	var multi *terpMulti
	if len(zz) == 1 && zz[0].Type == BARE {
		multi = zz[0].Multi
	}

	return &PWord{Parts: zz, ExpandAsMultiWord: me.ExpandAsMultiWord, Multi: multi}
}

func (me *PPart) CloneAndSubst(params map[string][]*PWord) []*PPart {
	switch me.Type {
	case BARE:
		return []*PPart{me}
	case DOLLAR1:
		if me.VarName == "ARGS" {
			panic("'$ARGS' cannot be used in this context, in part of a word, in a macro.")
		}
		if vec, ok := params[me.VarName]; ok {
			CHECK(len(vec) == 1, len(vec), vec)
			return vec[0].Parts
		} else {
			return []*PPart{me}
		}
	case DOLLAR2:
		vec := me.Word.CloneAndSubst(params)
		CHECK(len(vec) == 1, len(vec), vec, me)
		return []*PPart{&PPart{Type: DOLLAR2, VarName: me.VarName, Word: vec[0]}}
	case SQUARE:
		return []*PPart{&PPart{Type: SQUARE, Seq: me.Seq.CloneAndSubst(params)}}
	}
	panic("unknown PartType")
}

func (me *PPart) ExpandMacros(fr *Frame, maxSubCompile int) *PPart {
	switch me.Type {
	case BARE:
		if len(me.Multi.s.s) >= maxSubCompile {
			return me
		} else {
			return &PPart{Type: BARE, Multi: MkMultiFr(fr, me.Multi)}
		}
	case DOLLAR1:
		return me
	case DOLLAR2:
		return &PPart{Type: DOLLAR2, VarName: me.VarName, Word: me.Word.ExpandMacros(fr, maxSubCompile)}
	case SQUARE:
		return &PPart{Type: SQUARE, Seq: me.Seq.ExpandMacros(fr, maxSubCompile)}
	}
	panic("unknown PartType")
}

func CHECK(b bool, rest ...interface{}) {
	if !b {
		s := "CHECK FAILS"
		for _, e := range rest {
			s += Sprintf("; %v", e)
		}
		panic(s)
	}
}

/////////////////

var Parse2CmdCounter Counter
var Parse2DollarCounter Counter
var Parse2SquareCounter Counter
var Parse2QuoteCounter Counter
var Parse2SeqCounter Counter
var Parse2SeqEvalCounter Counter
var Parse2CmdEvalCounter Counter
var Parse2WordEvalFastCounter0 Counter
var Parse2WordEvalFastCounter1 Counter
var Parse2WordEvalSlowCounter1 Counter
var Parse2WordEvalSlowCounter9 Counter

var Parse2ExprTopCounter Counter
var Parse2ExprEvalCounter Counter

func init() {
	Parse2CmdCounter.Register("Parse2Cmd")
	Parse2DollarCounter.Register("Parse2Dollar")
	Parse2SquareCounter.Register("Parse2Square")
	Parse2QuoteCounter.Register("Parse2Quote")
	Parse2SeqCounter.Register("Parse2Seq")
	Parse2SeqEvalCounter.Register("Parse2SeqEval")
	Parse2CmdEvalCounter.Register("Parse2CmdEval")
	Parse2WordEvalFastCounter0.Register("Parse2WordEvalFast0")
	Parse2WordEvalFastCounter1.Register("Parse2WordEvalFast1")
	Parse2WordEvalSlowCounter1.Register("Parse2WordEvalSlow1")
	Parse2WordEvalSlowCounter9.Register("Parse2WordEvalSlow9")
	Parse2ExprTopCounter.Register("Parse2ExprTop")
	Parse2ExprEvalCounter.Register("Parse2ExprEval")
}
