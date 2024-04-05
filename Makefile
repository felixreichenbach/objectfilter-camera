bin/objectfilter:
	go build -ldflags '-X main.Version=$(shell git describe --tags) -extldflags "-static"' -o $@ ./src

module.tar.gz: bin/objectfilter
	tar czf $@ $<
