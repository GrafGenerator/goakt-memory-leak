.PHONY: modules.update lint lint-fix fmt mockery download tools gen swagger doc tests

bufgen:
	buf generate \
		--template buf.gen.yaml \
		--path protos/messages.proto

	cp -r gen/* internal/messages
