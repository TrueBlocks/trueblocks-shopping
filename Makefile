MSG ?= update

.PHONY: build test lint type-check clean clobber add commit push

build:
	yarn build

test:
	yarn test

lint:
	yarn lint

type-check:
	yarn type-check

clean:
	rm -rf build/bin

clobber: clean
	@find . \( -path './.git' -o -path './.git/*' \) -prune -o -type d -name node_modules -print -exec rm -rf {} +

add:
	@git add -A

commit: build
	@git add -A
	@git commit -m "$(MSG)" || true

push: build
	@git add -A
	@git commit -m "$(MSG)" || true
	@git push
