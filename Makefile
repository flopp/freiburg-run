all:
	@echo "make upload -> build and upload to freiburg.run"

.phony: build
build:
	rm -rf .out
	go run main.go

.phony: upload
upload: build
	scp -r .out/* echeclus.uberspace.de:/var/www/virtual/floppnet/freiburg.run/

.phony: upload-test
upload-test: build
	scp -r .out/* echeclus.uberspace.de:/var/www/virtual/floppnet/fraig.de/
