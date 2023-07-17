# SKR test

This test covers SAP BTP, Kyma runtime (SKR).

## Usage

Prepare the `.env` file based on the [.env.template](.env.template). Run the following command to set up the environment variables in your system:

```bash
export $(xargs < .env)
```

Run the test scenario:

```bash
npm run skr-aws-upgrade-integration-test
```

## Environment variables
**AL_SERVICE_KEY** must be a Cloud Foundry service key with the info about UAA (User Account and Authentication). Learn more about [Managing Service Keys in Cloud Foundry](https://docs.cloudfoundry.org/devguide/services/service-keys.html).
