all: demo

fmt:
	gofmt -w *.go */*.go

demo: _FORCE_
	for x in demo/*.tcl ; do echo == $$x == ; go run tcl67.go $$x ; done

_FORCE_:
