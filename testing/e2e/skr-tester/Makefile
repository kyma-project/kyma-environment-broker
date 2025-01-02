#
# Make targets to build skr-tester artifacts
#

ifndef ARTIFACTS
	ARTIFACTS = .
endif

build: $(foreach TOOL,skr-tester,$(foreach OS,linux windows darwin,build-$(OS)-$(TOOL)))

build-windows-%:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o $(ARTIFACTS)/$*.exe $(CLI_FLAGS) cmd/main.go

build-linux-%:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $(ARTIFACTS)/$*-linux $(CLI_FLAGS) cmd/main.go

build-darwin-%:
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o $(ARTIFACTS)/$*-darwin $(CLI_FLAGS) cmd/main.go
