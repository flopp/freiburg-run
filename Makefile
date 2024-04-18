all:
	@echo "make upload -> build and upload to freiburg.run"

.bin/generate-linux: main.go internal/utils/*.go go.mod
	mkdir -p .bin
	GOOS=linux GOARCH=amd64 go build -o .bin/generate-linux main.go

.phony: build
build:
	rm -rf .out
	go run main.go -config config.json -out .out -hashfile .hashes -addedfile .added

.phony: upload
upload: build
	scp -r .out/* .out/.htaccess echeclus.uberspace.de:/var/www/virtual/floppnet/freiburg.run/
	scp -r .out/* .out/.htaccess echeclus.uberspace.de:/var/www/virtual/floppnet/fraig.de/

.phony: upload-test
upload-test: build
	scp -r .out/* .out/.htaccess echeclus.uberspace.de:/var/www/virtual/floppnet/fraig.de/
	ssh echeclus.uberspace.de chmod -R a+rx /var/www/virtual/floppnet/fraig.de/

.repo/.git/config:
	git clone https://github.com/flopp/freiburg-run.git .repo

.phony: sync
sync: .repo/.git/config .bin/generate-linux
	(cd .repo && git pull --quiet)
	rsync -a scripts/cronjob.sh .bin/generate-linux echeclus.uberspace.de:packages/freiburg.run/
	rsync -a .repo/ echeclus.uberspace.de:packages/freiburg.run/repo
	ssh echeclus.uberspace.de chmod +x packages/freiburg.run/cronjob.sh packages/freiburg.run/generate-linux

.phony: run-script
run-script: sync
	ssh echeclus.uberspace.de packages/freiburg.run/cronjob.sh

.phony: lint
lint:
	go vet ./...