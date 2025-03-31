# HAP Parser

This folder contains the sources of the tools for verifying the correctness of the Hyperscaler Account Pool (HAP) configuration.

### Build Tool

Run the following command to build the binary:

```
make build-hap
```

Executable file `hap` will be created in the `./bin` directory.

### Running

Run the following command to show the help message for the `parse` command:
```
./bin/hap parse -h
```

### Examples

Run the following command to verify the correctness of the HAP configuration and check which rule will be matched given the provisioning data:
```
./bin/hap parse  -e 'aws;gcp'  -m '{"plan": "aws", "platformRegion": "cf-eu11", "hyperscalerRegion": "westeurope", "hyperscaler":"aws"}'
Your rule configuration is OK.
Matched rule: aws
```

Check correctness of the HAP configuration in the file 'rules/rules-final.yaml':
```shell
./bin/hap parse -f cmd/parser/rules/rules-final.yaml
```