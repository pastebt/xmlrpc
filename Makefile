test:
	@echo "test xmlrpc"
	@go test -v -coverprofile /tmp/x.out
	@go tool cover -html=/tmp/x.out -o /tmp/x.html
	@rm /tmp/x.out
