APP_NAME = rodeo
SSH_REMOTE = sierra

dev:
	@which reflex>/dev/null || go install github.com/cespare/reflex@latest
	reflex -s -d none -R '(node_modules|contracts)' -r '\.(html|js|css|go)$$' -- make start

start:
	go run *.go start

run:
	env SECRET=$(RODEO_SECRETS_PASSWORD_DEVELOPMENT) go run *.go $(filter-out $@,$(MAKECMDGOALS))

run-prod:
	env ENV=production SECRET=$(RODEO_SECRETS_PASSWORD_PRODUCTION) go run *.go $(filter-out $@,$(MAKECMDGOALS))

build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -ldflags "-s -w" -installsuffix cgo -o server *.go

deploy: build
	scp server $(SSH_REMOTE):/tmp/dtmpl
	ssh $(SSH_REMOTE) "mv /tmp/dtmpl /root/$(APP_NAME)"
	ssh $(SSH_REMOTE) "sudo systemctl restart $(APP_NAME)"

ssh:
	ssh $(SSH_REMOTE)

logs:
	ssh $(SSH_REMOTE) "journalctl -n 500 -f -u rodeo"

deployCustom:
  env forge script Deploy.s.sol:Deploy \
      --rpc-url "https://rpc.tenderly.co/fork/f03d0d80-0f91-4f80-bfb5-eab2876ca861"  \
      --private-key "04299b9939d3f947130394edbdba609f47273db7cc7a75aa309c3636eee739f7" \
      --broadcast -vvvv

%:
	@:
