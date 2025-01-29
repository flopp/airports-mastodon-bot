.bin/bot-linux: cmd/bot/main.go internal/airports/*.go internal/data/*.go go.mod
	mkdir -p .bin
	GOOS=linux GOARCH=amd64 go build -o .bin/bot-linux cmd/bot/main.go

.PHONY: deploy
deploy: .bin/bot-linux
	ssh echeclus.uberspace.de mkdir -p packages/airports-mastodon-bot
	scp -r production-config.json scripts/cronjob.sh .bin/bot-linux .data echeclus.uberspace.de:packages/airports-mastodon-bot
	ssh echeclus.uberspace.de chmod +x packages/airports-mastodon-bot/cronjob.sh packages/airports-mastodon-bot/bot-linux

.PHONY: run-bot
run-bot:
	ssh echeclus.uberspace.de packages/airports-mastodon-bot/cronjob.sh