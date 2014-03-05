PROG_NAME=cmm

default: dependencies
	go build -o bin/$(PROG_NAME)

dependencies:
	go list -f "{{range .Imports}}{{.}} {{end}}" ./ | xargs go get

todo:
	grep -nri "// TODO:"

install:
	cp bin/$(PROG_NAME) /usr/local/bin/

.PHONY: todo install