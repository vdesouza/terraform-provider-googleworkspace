SWEEP?=us-central1
TEST?=$$(go list ./...)

default: build

build: fmtcheck
	go install

fmt:
	@echo "==> Fixing source code with gofmt..."
	gofmt -w -s ./internal/provider

# Currently required by tf-deploy compile
fmtcheck:
	@echo "==> Checking source code against gofmt..."
	@sh -c "'$(CURDIR)/scripts/gofmtcheck.sh'"

generate: build
	go generate  ./...

lint:
	@echo "==> Checking source code against linters..."
	@golangci-lint run ./internal/provider

sweep:
	@echo "WARNING: This will destroy infrastructure. Use only in development accounts."
	go test ./internal/provider -v -sweep=$(SWEEP) -sweep-run=$(SWEEPARGS) -timeout 60m

test: fmtcheck
	go test -count=1 $(TESTARGS) -timeout=30s $(TEST)

# Run acceptance tests
.PHONY: testacc
testacc: fmtcheck
	TF_ACC=1 go test -count=1 $(TEST) -v $(TESTARGS) -timeout 120m

# Validate companion modules under modules/
.PHONY: validate-modules
validate-modules:
	@for d in modules/*/; do \
	  echo "==> terraform init/validate $$d"; \
	  (cd $$d && terraform init -backend=false -upgrade && terraform validate) || exit 1; \
	done

# Validate runnable examples under examples/modules/
.PHONY: validate-examples
validate-examples:
	@for d in examples/modules/*/; do \
	  [ -f $$d/main.tf ] || continue; \
	  echo "==> terraform init/validate $$d"; \
	  (cd $$d && terraform init -backend=false -upgrade && terraform validate) || exit 1; \
	done

# Format Terraform files under modules/ and examples/
.PHONY: fmt-tf
fmt-tf:
	terraform fmt -recursive ./modules/ ./examples/