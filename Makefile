-include .env

.PHONY: demo-migrate demo-seed demo-scan demo-complete demo-verify demo-dunning-flow

demo-migrate:
	go run ./cmd/demo migrate

demo-seed:
	go run ./cmd/demo seed

demo-scan:
	go run ./cmd/demo scan

demo-complete:
	go run ./cmd/demo complete -token "$(TOKEN)"

demo-verify:
	go run ./cmd/demo verify

demo-dunning-flow:
	./scripts/demo_dunning_flow.sh
