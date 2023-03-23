package cli

/*
	For debugging exec pipes, try this:
	go run chirp.go -recover=0 -c='puts [exec ls -l | sed {s/[0-9]/#/g} | tr {a-z} {A-Z} ]' 2>/dev/null | od -c
*/

import (
	"github.com/strickyak/tcl67/tcl"

	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
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

func saveArgvStarting(fr *tcl.Frame, n int) {
	argv := []tcl.T{}
	for i, a := range flag.Args() {
		if i >= n {
			argv = append(argv, tcl.MkString(a))
		}
	}
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

	if *cFlag != "" {
		saveArgvStarting(fr, 0)
		fr.Eval(tcl.MkString(*cFlag))
		goto End
	}

	if len(flag.Args()) > 0 {
		// Script mode.
		scriptName = flag.Arg(0)
		contents, err := ioutil.ReadFile(scriptName)
		if err != nil {
			log.Fatalf("Cannot read file %s: %v", scriptName, err)
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
			HistoryFile:     filepath.Join(home, ".tcl67.history"),
			InterruptPrompt: "*SIGINT*",
			EOFPrompt:       "*EOF*",
			// AutoComplete:    completer,
			// HistorySearchFold:   true,
			// FuncFilterInputRune: filterInput,
		})
		if err != nil {
			log.Fatalf("Cannot create readline object: %v", err)
		}
		defer rl.Close()

		i := 1
		for {
			line, err := rl.Readline()
			if err != nil {
				if err != io.EOF {
					log.Fatalf("*** ERROR in Readline: %s\n", err.Error())
				}
				goto End
			}
			result := EvalStringOrPrintError(fr, string(line))
			resultStr := result.String()
			if resultStr != "" { // Traditionally, if result is empty, tclsh doesn't print.
				fmt.Printf("$%d = %s\n", i, resultStr)
				fr.SetVar(tcl.Str(i), result)
				i++
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
			log.Fatalf("TEST FAILS: %q succeeds=%d fails=%d\n", scriptName, tcl.MustSucceeds, tcl.MustFails)
		}
		log.Printf("Test Done: %q succeeds=%d\n", scriptName, tcl.MustSucceeds)
		tcl.MustMutex.Unlock()
	}
}

func EvalStringOrPrintError(fr *tcl.Frame, cmd string) (out tcl.T) {
	if *recoverFlag {
		defer func() {
			if r := recover(); r != nil {
				fmt.Fprintln(os.Stderr, "ERROR: ", r) // Error to stderr.
				out = tcl.Empty
				return
			}
		}()
	}

	return fr.Eval(tcl.MkString(cmd))
}
