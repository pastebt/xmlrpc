test:
	@echo "test xmlrpc"
	@go test -v -coverprofile /tmp/x.out
	@#sed s%"^_/home/yma/svn/Webfilter/trunk/fure2go/fu2g"%"\."% c.out > c.out.g
	@go tool cover -html=/tmp/x.out -o /tmp/x.html
	@rm /tmp/x.out
