chineseWordsCnt := $(shell find . -iname "*.md" -print0 | grep -z -v _book | grep -z -v node_modules |  wc -m --files0-from - | tail -n 1 | cut -f1)
#englishWordsCnt := $(shell find . -iname "*.md" -print0 | grep -z -v zh_CN | grep -z -v _book | grep -z -v node_modules |  wc -m --files0-from - | tail -n 1 | cut -f1)

stat:
	@echo "Chinese version, words: ${chineseWordsCnt}"
	#@echo "English version, words: ${englishWordsCnt}"
