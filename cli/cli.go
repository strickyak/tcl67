package cli

/*
	For debugging exec pipes, try this:
	go run chirp.go -recover=0 -c='puts [exec ls -l | sed {s/[0-9]/#/g} | tr {a-z} {A-Z} ]' 2>/dev/null | od -c
*/

import (
	"github.com/strickyak/tcl67/tcl"

	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime/pprof"
	"strings"

	"github.com/chzyer/readline"
)

var dFlag = flag.String("d", "", "Debugging flags, each a single letter.")
var cFlag = flag.String("c", "", "Immediate command to execute.")
var recoverFlag = flag.Bool("recover", true, "Set to false to disable recover in the REPL.")
var testFlag = flag.Bool("test", false, "Print test summary at end.")

var scriptName string

func saveArgvStarting(fr *tcl.Frame, i int) {
	argv := []tcl.T{}
	for _, a := range flag.Args() {
		argv = append(argv, tcl.MkString(a))
	}
	fr.SetVar("argv", tcl.MkList(argv)) // Deprecated: argv
	fr.SetVar("Argv", tcl.MkList(argv)) // New: Argv
}

func setEnvironInChirp(fr *tcl.Frame, varName string) {
	h := make(tcl.Hash)
	for _, s := range os.Environ() {
		kv := strings.SplitN(s, "=", 2)
		if len(kv) == 2 {
			h[kv[0]] = tcl.MkString(kv[1])
		}
	}
	fr.SetVar(varName, tcl.MkHash(h))
}

func Main() {
	flag.Parse()
	fr := tcl.NewInterpreter()
	setEnvironInChirp(fr, "Env")

	for _, ch := range *dFlag {
		if ch < 256 {
			tcl.Debug[ch] = true
		}
	}

	if cFlag != nil && *cFlag != "" {
		saveArgvStarting(fr, 1)
		fr.Eval(tcl.MkString(*cFlag))
		goto End
	}

	if len(flag.Args()) > 0 {
		// Script mode.
		scriptName = flag.Arg(0)
		contents, err := ioutil.ReadFile(scriptName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Cannot read file %s: %v", scriptName, err)
			os.Exit(2)
			return
		}
		saveArgvStarting(fr, 1)

		fr.Eval(tcl.MkString(string(contents)))
		goto End
	}

	{
		// Interactive mode.
		home := os.Getenv("HOME")
		if home == "" {
			home = "."
		}

		rl, err := readline.NewEx(&readline.Config{
			Prompt:          "% ",
			HistoryFile:     filepath.Join(home, ".tcl.history"),
			InterruptPrompt: "*SIGINT*",
			EOFPrompt:       "*EOF*",
			// AutoComplete:    completer,
			// HistorySearchFold:   true,
			// FuncFilterInputRune: filterInput,
		})
		if err != nil {
			panic(err)
		}
		defer rl.Close()

		for {
			fmt.Fprint(os.Stderr, "tcl% ") // Prompt to stderr.
			line, err := rl.Readline()
			if err != nil {
				if err.Error() == "EOF" { // TODO: better way?
					goto End
				}
				fmt.Fprintf(os.Stderr, "ERROR in Readline: %s\n", err.Error())
				goto End
			}
			result := EvalStringOrPrintError(fr, string(line))
			if result != "" { // Traditionally, if result is empty, tclsh doesn't print.
				fmt.Println(result)
			}
		}
	}

End:
	logAllCounters()
	if tcl.Debug['h'] {
		pprof.Lookup("heap").WriteTo(os.Stderr, 0)
	}
}

func logAllCounters() {
	if tcl.Debug['c'] {
		tcl.LogAllCounters()
	}

	// Print summary for tests.
	if *testFlag {
		tcl.MustMutex.Lock()
		if tcl.MustFails > 0 {
			fmt.Fprintf(os.Stderr, "TEST FAILS: %q succeeds=%d fails=%d\n", scriptName, tcl.MustSucceeds, tcl.MustFails)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Test Done: %q succeeds=%d\n", scriptName, tcl.MustSucceeds)
		tcl.MustMutex.Unlock()
	}
}

func EvalStringOrPrintError(fr *tcl.Frame, cmd string) (out string) {
	if *recoverFlag {
		defer func() {
			if r := recover(); r != nil {
				fmt.Fprintln(os.Stderr, "ERROR: ", r) // Error to stderr.
				out = ""
				return
			}
		}()
	}

	return fr.Eval(tcl.MkString(cmd)).String()
}
