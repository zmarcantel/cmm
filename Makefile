PROG_NAME=cmm

default: dependencies
	go build -o bin/$(PROG_NAME)

dependencies:
	go list -f "{{ range .Deps }}{{ . }} {{ end }}" ./ | tr ' ' '\n' | awk '!/^.\//' | xargs go get

todo:
	grep -nri "TODO:"

install: default
	cp bin/$(PROG_NAME) /usr/local/bin/

vm:
	vagrant up

test:
	vagrant provision

.PHONY: todo install vm
