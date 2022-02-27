lines=$(shell cat `find -iname "*.go" | grep -vE '_fixtures|_scripts|_testcase|_test.go'` | wc -l)
linesIncludeTest=$(shell cat `find -iname "*.go" | grep -vE '_fixtures|_scripts|_testcase'` | wc -l)

stat:
	@echo "lines without test: ${lines}"
	@echo "lines include test: ${linesIncludeTest}"
