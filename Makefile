MSG ?= update

.PHONY: build test lint clean add commit push

build:
	yarn build

test:
	yarn test

lint:
	yarn lint

clean:
	rm -rf build/bin

add:
	@git add -A

commit: build
	@git add -A
	@git commit -m "$(MSG)" || true

push: build
	@git add -A
	@git commit -m "$(MSG)" || true
	@git push
