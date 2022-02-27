lines=$(shell cat `find -iname "*.go" | grep -vE '_fixtures|_scripts|testcase|_test.go'` | wc -l)
linesIncludeTest=$(shell cat `find -iname "*.go" | grep -vE '_fixtures|_scripts|testcase'` | wc -l)

stat:
	@echo "lines without test: ${lines}"
	@echo "lines include test: ${linesIncludeTest}"
