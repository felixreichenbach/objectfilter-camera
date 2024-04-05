bin/objectfilter:
	go build -ldflags '-X main.Version=$(shell git describe --tags)' -tags osusergo,netgo -o $@ ./src

module.tar.gz: bin/objectfilter
	tar czf $@ $<
