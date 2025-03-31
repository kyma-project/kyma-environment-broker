# HAP Parser

This folder contains the sources of the tools for verifying the correctness of the Hyperscaler Account Pool (HAP) configuration.

### Build Tool

To build the binary, run the following command:

```
make build-hap
```

The executable `hap` file is created in the `./bin` directory.

### Running

To show the help message for the `parse` command, run:
```
./bin/hap parse -h
```

### Examples

To verify the correctness of the HAP configuration and check which rule is matched given the provisioning data, run the following command:
```
./bin/hap parse  -e 'aws;gcp'  -m '{"plan": "aws", "platformRegion": "cf-eu11", "hyperscalerRegion": "westeurope", "hyperscaler":"aws"}'
Your rule configuration is OK.
Matched rule: aws
```

Check correctness of the HAP configuration in the file 'rules/rules-final.yaml':
```shell
./bin/hap parse -f cmd/parser/rules/rules-final.yaml
```