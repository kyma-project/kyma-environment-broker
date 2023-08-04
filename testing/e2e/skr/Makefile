.PHONY: lint
lint:
	npm install
	npm run lint

.PHONY: lint-fix
lint-fix:
	npm install
	npm run lint-fix

.PHONY: keb-endpoints
keb-endpoints:
	npm install
	npm run keb-endpoints-test

.PHONY: skr
skr:
	curl -fLSs -o /usr/local/bin/kcp https://storage.googleapis.com/kyma-development-artifacts/kcp/master/kcp-linux
	chmod +x /usr/local/bin/kcp
	npm install
	npm run skr-test

.PHONY: skr-aws-upgrade-integration
skr-aws-upgrade-integration:
	npm install
	npm run skr-aws-upgrade-integration-test
