default:
	go build -o bin/cmm

todo:
	grep -nri "// TODO:"

.PHONY: todo