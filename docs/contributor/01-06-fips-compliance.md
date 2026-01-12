# FIPS 140-3 Compliance

This document describes the current FIPS 140-3 compliance of the Kyma Environment Broker (KEB).

## Current Status

KEB is compiled and executed in the FIPS 140-3 mode using the Go FIPS module and runtime controls. The build and runtime configuration is defined in the KEB’s [Dockerfile](../../Dockerfile.keb).

### Build-Time controls

KEB is built with the Go FIPS module enabled. The Dockerfile has a **build** stage where the **RUN** command includes the `GOFIPS140=v1.0.0` environment variable:

```Dockerfile
RUN --mount=type=cache,target=/root/.cache/go-build \
		--mount=type=cache,target=/go/pkg \
		CGO_ENABLED=0 GOFIPS140=v1.0.0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
		go build -ldflags "-X main.Version=${VERSION}" \
		-o /bin/kyma-env-broker ./cmd/broker/
```

The `GOFIPS140=v1.0.0` environment variable enables building against the Go FIPS 140 module and configures the application to run in the FIPS 140-3 mode.

### Runtime Controls

The final image enforces FIPS-only operation through the **GODEBUG** environment variable:

```Dockerfile
ENV GODEBUG=fips140=only,tlsmlkem=0
```

The `fips140=only` option restricts cryptographic operations to those approved by the Go FIPS module. The `tlsmlkem=0` option disables the post-quantum key exchange mechanism X25519MLKEM768 to avoid usage of not FIPS-approved algorithms.

This combination ensures the broker runs exclusively with FIPS-approved algorithms provided by Go’s FIPS module.

## Approved Cryptographic Algorithms

For the authoritative list and requirements around approved algorithms and testing, refer to [SP 800-140C: Approved Security Functions](https://csrc.nist.gov/projects/cmvp/sp800-140c).

When `GODEBUG=fips140=only` is active, KEB relies on the Go FIPS module’s set of approved algorithms. Any attempt to use non-approved algorithms will be blocked at the runtime.

## PostgreSQL Password Authentication Requirements

When connecting to PostgreSQL, configure password authentication to use SCRAM-SHA-256 and avoid deprecated MD5. For the required configuration steps, see [PostgreSQL documentation](https://www.postgresql.org/docs/current/auth-password.html).

The minimal password length is 112 bits.

> [!TIP]
> _To upgrade an existing installation from `md5` to `scram-sha-256`, after having ensured that all client libraries in use are new enough to support SCRAM, set `password_encryption = 'scram-sha-256'` in `postgresql.conf`, make all users set new passwords, and change the authentication method specifications in `pg_hba.conf` to `scram-sha-256`._ (From the [PostgreSQL documentation](https://www.postgresql.org/docs/current/auth-password.html)).

## Verification

- Build logs should show `GOFIPS140=v1.0.0` in the Go build environment.
- The running container must include `GODEBUG=fips140=only,tlsmlkem=0` in its environment. You can confirm it using your container runtime inspection tooling.
- PostgreSQL should report SCRAM-based authentication in logs and reject MD5-based attempts.