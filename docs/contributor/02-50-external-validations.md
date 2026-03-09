# External Validations: Kyma Instances Quota

Each service or environment is responsible for managing its own quota usage. During provisioning requests, Kyma Environment Broker (KEB) initiates a call 
to the Entitlements Service to retrieve the assigned quota for the target subaccount and plan. Such calls are also made during update requests if the plan changes.
If the assigned quota is less than or equal to the number of instances stored in the database, the request fails. 

The quota check is performed during provisioning if there is more than one Kyma environment per subaccount and the subaccount ID is not whitelisted.
During update requests, the quota check is not performed if the subaccount ID is whitelisted. If the request to the Entitlements Service fails, it is retried at configured intervals. 
If the retries are unsuccessful, the provisioning or update request is rejected.

The following configuration enables quota limit checks and specifies the required URLs, credentials, and retry behavior. 
Whitelisted subaccount IDs are excluded from quota validation.

```yaml
cis:
  entitlements:
    authURL: "https://authentication-service.com"
    id: "client-id"
    secret: "client-secret"
    secretName: "cis-creds-entitlements"
    serviceURL: "https://entitlements-service.com"
    clientIdKey: id
    secretKey: secret
quotaLimitCheck:
  enabled: true
  interval: 1s
  retries: 5
quotaWhitelistedSubaccountIds: |-
  whitelist:
    - whitelisted-subaccount-1
    - whitelisted-subaccount-2
```
