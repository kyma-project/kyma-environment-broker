# Configure list of modules

By default, Kyma Environment Broker (KEB) applies the Kyma custom resource, including the default modules selected by Kyma, to a cluster.
You can configure your custom list of modules by setting the **modules** object in the JSON schema.
To do that, use the following two fields:
- **default** (bool) - defines whether to use the default list of modules
- **list** (array) - defines a custom list of modules

API for module configuration is built on the **oneOf** feature from the JSON schema. If the **modules** object is passed to API, it must have only one valid option. Thus, to pass JSON API Validator, you must set only one field. See examples below.

## Correct API calls

- In the default mode, the **modules** object is set to `default: true` and modules are pre-selected by Kyma. The same happens when the **modules** section is not set at all (mapped to nil) in the payload.

   ```
   "modules": {
       "default": true
   }
   ```

- If you want to use your custom modules' list configuration, you must pass the **list** with the modules you want installed.

   ```
   "modules": {
       "list": [
           {
               "name": "btp-operator",
               "customResourcePolicy": "CreateAndDelete"
           },
           {
               "name": "keda",
               "channel": "fast",
           }
       ]
   }
   ```

- The following calls result in Kyma runtime installation without any modules.

   ```
   "modules": {
       "list": []
   }
   ```

   ```
   "modules": {
       "default": false
   }
   ```

## Incorrect API calls

- A call with the **modules** section empty fails due to using the **oneOf** feature in the JSON schema.

   ```
   "modules": {}
   ```

- A call with the **modules** section with both parameters filled in fails due to using the **oneOf** feature in the JSON schema.

   ```
   "modules": {
       "default": false,
       "list": [
           {
               "name": "btp-operator",
               "customResourcePolicy": "CreateAndDelete"
           },
           {
               "name": "keda",
               "channel": "fast",
           }
       ]
   }
   ```
