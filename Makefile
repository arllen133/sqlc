.PHONY: gen-examples

gen-examples:
	@for dir in examples/*; do \
		if [ -d "$$dir/models" ]; then \
			echo "Generating for $$dir..."; \
			go run cmd/sqlcli/main.go -i $$dir/models; \
		fi \
	done
