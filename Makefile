all:
	@echo "make upload -> build and upload to freiburg.run"

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
sync: .repo/.git/config
	(cd .repo && git pull --quiet)
	rsync -a .repo/ echeclus.uberspace.de:packages/freiburg.run/repo

.phony: run-script
run-script: sync
	ssh echeclus.uberspace.de packages/freiburg.run/cronjob.sh
