.PHONY: gen-proto

gen-proto:
	cd proto && buf generate
