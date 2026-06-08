.PHONY: install test mutate mutate-html

install:
	go install ./cmd/mutagen/

test:
	go test ./pkg/... -count=1

mutate: install
	mutagen -v -workers 4 -no-cache ./pkg/...

mutate-html: install
	mutagen -v -workers 4 -no-cache -html report.html ./pkg/...
