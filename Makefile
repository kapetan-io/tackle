.DEFAULT_GOAL := build
LINT = $(GOPATH)/bin/golangci-lint
LINT_VERSION = 1.56.2

$(LINT): ## Download Go linter
	curl -sfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(GOPATH)/bin $(LINT_VERSION)

.PHONY: test
test:
	go test -timeout 10m -v -p=1 -count=1 -race -tags clock_mutex ./...

.PHONY: lint
lint: $(LINT) ## Run Go linter
	$(LINT) run -v ./...

.PHONY: tidy
tidy:
	go mod tidy && git diff --exit-code

.PHONY: ci
ci: tidy lint test
	@echo
	@echo "\033[32mEVERYTHING PASSED!\033[0m"


.PHONY: certs
certs: ## Generate SSL certificates
	rm autotls/certs/*.key || rm autotls/certs/*.srl || rm autotls/certs/*.csr || rm autotls/certs/*.pem || rm autotls/certs/*.cert || true
	openssl genrsa -out autotls/certs/ca.key 4096
	openssl req -new -x509 -key autotls/certs/ca.key -sha256 -subj "/C=US/ST=TX/O=Opensource" -days 3650 -out autotls/certs/ca.cert
	openssl genrsa -out autotls/certs/auto.key 4096
	openssl req -new -key autotls/certs/auto.key -out autotls/certs/auto.csr -config autotls/certs/auto.conf
	openssl x509 -req -in autotls/certs/auto.csr -CA autotls/certs/ca.cert -CAkey autotls/certs/ca.key -set_serial 1 -out autotls/certs/auto.pem -days 3650 -sha256 -extfile autotls/certs/auto.conf -extensions req_ext
	openssl genrsa -out autotls/certs/auto_no_ip_san.key 4096
	openssl req -new -key autotls/certs/auto_no_ip_san.key -out autotls/certs/auto_no_ip_san.csr -config autotls/certs/auto_no_ip_san.conf
	openssl x509 -req -in autotls/certs/auto_no_ip_san.csr -CA autotls/certs/ca.cert -CAkey autotls/certs/ca.key -set_serial 2 -out autotls/certs/auto_no_ip_san.pem -days 3650 -sha256 -extfile autotls/certs/auto_no_ip_san.conf -extensions req_ext
	# Client Auth
	openssl req -new -x509 -days 3650 -keyout autotls/certs/client-auth-ca.key -out autotls/certs/client-auth-ca.pem -subj "/C=TX/ST=TX/O=Opensource/CN=auto.io/emailAddress=admin@auto-rpc.org" -passout pass:test
	openssl genrsa -out autotls/certs/client-auth.key 2048
	openssl req -sha1 -key autotls/certs/client-auth.key -new -out autotls/certs/client-auth.req -subj "/C=US/ST=TX/O=Opensource/CN=client.com/emailAddress=admin@auto-rpc.org"
	openssl x509 -req -days 3650 -in autotls/certs/client-auth.req -CA autotls/certs/client-auth-ca.pem -CAkey autotls/certs/client-auth-ca.key -set_serial 3 -passin pass:test -out autotls/certs/client-auth.pem
	openssl x509 -extfile autotls/certs/client-auth.conf -extensions ssl_client -req -days 3650 -in autotls/certs/client-auth.req -CA autotls/certs/client-auth-ca.pem -CAkey autotls/certs/client-auth-ca.key -set_serial 4 -passin pass:test -out autotls/certs/client-auth.pem
