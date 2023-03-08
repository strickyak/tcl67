# tcl67
An extensible Tcl interpreter written in Go, at about the level of Tcl 6.7, with some differences. 

( based on earlier work: https://github.com/yak-labs/chirp-lang )

## Try This

`for x in demo/*.tcl ; do echo == $x == ; go run tcl67.go $x ; done`
