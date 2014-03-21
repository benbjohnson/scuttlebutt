default: build

build: assets templates
	mkdir -p build
	cd cmd/scuttlebutt && goxc -c=.goxc.json -pr="$(PRERELEASE)" -d ../../build
