PROG_NAME=cmm

default:
	go get ./
	go build -o bin/$(PROG_NAME)

todo:
	grep -nri "// TODO:"

install:
	cp bin/$(PROG_NAME) /usr/local/bin/

.PHONY: todo install