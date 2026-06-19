<!--{"metadata":{"publish":false}}-->
# Zone Discovery — Architecture Diagrams

## 1. Full provisioning flow

```mermaid
sequenceDiagram
    participant C as OSB Client
    participant P as ProvisionEndpoint
    participant G as Gardener (K8s)
    participant F as ClientFactory
    participant A as Cloud API (AWS/Azure)
    participant Q as Worker Queue

    C->>P: POST /v2/service_instances
    Note over P: Parse params<br/>provider, region, machineType

    P->>G: GetCredentialsBindings(selector)
    G-->>P: CredentialsBinding → Secret ref

    P->>G: GetSecret(namespace, name)
    G-->>P: Secret {accessKeyID, secretAccessKey}<br/>or {clientID, tenantID, ...}

    P->>F: factory.NewFromSecret(secret, region)
    F-->>P: client

    P->>A: client.AvailableZones(machineType)
    Note over A: AWS: DescribeInstanceTypeOfferings<br/>Azure: ResourceSKUs.List
    A-->>P: ["eu-central-1a", "eu-central-1b", "eu-central-1c"]

    Note over P: Validate zone count ≥ required

    P->>Q: Enqueue operation

    Q->>G: DiscoverAvailableZonesCBStep<br/>(same flow, async)
    Q->>G: CreateRuntimeResourceStep<br/>Shoot CR with zones
```

---

## 2. Interface hierarchy — current vs proposed

```mermaid
classDiagram
    class FactoryRegistry {
        <<interface — PROPOSED>>
        +Factory(provider CloudProvider) ClientFactory, error
    }

    class ClientFactory {
        <<interface — EXISTS>>
        +NewFromSecret(ctx, secret, region) Client, error
    }

    class Client {
        <<interface — EXISTS>>
        +AvailableZones(ctx, machineType) []string, error
        +AvailableZonesCount(ctx, machineType) int, error
    }

    class AWSClientFactory {
        +NewFromSecret(ctx, secret, region) Client, error
        -extractCredentials(secret) accessKeyID, secretAccessKey
    }

    class AzureClientFactory {
        +NewFromSecret(ctx, secret, region) Client, error
        -extractCredentials(secret) clientID, clientSecret, tenantID, subscriptionID
    }

    class AWSClient {
        -ec2Client EC2API
        +AvailableZones(ctx, machineType) []string, error
        +AvailableZonesCount(ctx, machineType) int, error
    }

    class AzureClient {
        -skusClient ResourceSKUsClient
        +AvailableZones(ctx, machineType) []string, error
        +AvailableZonesCount(ctx, machineType) int, error
    }

    FactoryRegistry ..> ClientFactory : returns
    ClientFactory <|.. AWSClientFactory : implements
    ClientFactory <|.. AzureClientFactory : implements
    Client <|.. AWSClient : implements
    Client <|.. AzureClient : implements
    AWSClientFactory ..> AWSClient : creates
    AzureClientFactory ..> AzureClient : creates
```

---

## 3. Current map vs FactoryRegistry

```mermaid
flowchart LR
    subgraph current["CURRENT — map"]
        direction TB
        M["map[CloudProvider]ClientFactory\n{ AWS: awsFactory\n  Azure: azureFactory }"]
        MC["factory, ok := clientFactories[provider]\nif !ok { // bool check }"]
        M --> MC
    end

    subgraph proposed["PROPOSED — FactoryRegistry"]
        direction TB
        R["FactoryRegistry\n{ AWS: awsFactory\n  Azure: azureFactory }"]
        RC["factory, err := registry.Factory(provider)\nif err != nil { // error check }"]
        R --> RC
    end

    subgraph shared["SHARED — unchanged"]
        direction TB
        S1["factory.NewFromSecret(ctx, secret, region)"]
        S2["client.AvailableZones(ctx, machineType)"]
        S3["→ ['eu-central-1a', 'eu-central-1b', 'eu-central-1c']"]
        S1 --> S2 --> S3
    end

    current --> shared
    proposed --> shared
```

---

## 4. Prod vs Test wiring (proposed)

```mermaid
flowchart TD
    subgraph prod["Production"]
        NP["NewProdRegistry(providerSpec)"]
        NP --> PA["pkg.AWS → aws.NewFactory(spec)"]
        NP --> PZ["pkg.Azure → azure.NewFactory(spec)"]
    end

    subgraph test["Tests"]
        NT["NewTestRegistry(fixtures)"]
        NT --> TA["pkg.AWS → FakeAWSFactory(zones)"]
        NT --> TZ["pkg.Azure → FakeAzureFactory(zones)"]
    end

    subgraph usage["Step / Handler — identical code"]
        U["factory, err := registry.Factory(provider)\nclient, _ := factory.NewFromSecret(secret, region)\nzones, _ := client.AvailableZones(machineType)"]
    end

    prod --> usage
    test --> usage
```
