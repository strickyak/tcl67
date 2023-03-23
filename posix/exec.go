package posix

import (
	"log"
	"os/exec"

	. "github.com/strickyak/tcl67/tcl"
)

func cmdExec(fr *Frame, argv []T) T {
	commandT, argsT := Arg1v(argv)
	command := commandT.String()
	var args []string
	for _, a := range argsT {
		args = append(args, a.String())
	}

	cmd := exec.Command(command, args...)
	/*
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			log.Panicf("Cannot create StdoutPipe for command %q: %v", command, err)
		}
	*/

	bb, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			log.Panicf("Error in command %q: %q", command, ee.Stderr)
		}
		log.Panicf("Error in command %q: %v", command, err)
	}

	return MkString(string(bb))

	/*
		if err := cmd.Start(); err != nil {
			log.Panicf("Cannot Start command %q: %v", command, err)
		}

		var outbuf bytes.Buffer
		for {
			b := make([]byte, 4096)
			cc, err := stdout.Read(b)
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Panicf("Cannot read stdout of command %q: %v", command, err)
			}
			if cc == 0 {
			}
			outbuf.Write(b[:cc])
		}

		if err := cmd.Wait(); err != nil {
			log.Panicf("Bad Wait of command %q: %v", command, err)
		}
	*/
}

func init() {
	if Unsafes == nil {
		Unsafes = make(map[string]Command)
	}

	Unsafes["exec"] = cmdExec
}
