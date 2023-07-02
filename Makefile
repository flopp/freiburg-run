all:
	@echo "make upload -> build and upload to freiburg.run"

.phony: build
build:
	rm -rf .out
	go run main.go -config config.json

.phony: upload
upload: build
	scp -r .out/* .out/.htaccess echeclus.uberspace.de:/var/www/virtual/floppnet/freiburg.run/
	scp -r .out/* .out/.htaccess echeclus.uberspace.de:/var/www/virtual/floppnet/fraig.de/

.phony: upload-test
upload-test: build
	scp -r .out/* .out/.htaccess echeclus.uberspace.de:/var/www/virtual/floppnet/fraig.de/
