.PHONY: t
t:
	go test ./... -count=1 -race
