# Availability Zones in Kyma Environment

The Kyma environment uses availability zones to provide high availability and fault tolerance for your applications and workloads.

Availability zones are isolated failure domains within a single geographical region, connected through low-latency networks, and operating independently with their own power, network, and cooling infrastructure.

## Setup and Configurations

### Multi-Zone Architecture

SAP BTP, Kyma runtime operates on a three-availability-zone architecture. This multi-zone configuration enhances resilience against infrastructure failures by deploying your Kyma cluster, with its control plane and worker nodes, across multiple zones. Key aspects include the following:

- Distribution of Kubernetes worker nodes and control plane across three availability zones.
- Critical Kyma components, such as eventing, are configured in multiple zones.
- NAT gateways are provisioned to ensure a multi-zone setup.

The platform automatically manages node and zone failures for all managed components, including the API server (the seed).<!--???-->

The standard enterprise plans<!--or all plans except azure_lite???-->, `aws`, `gcp`, and `azure`, offer highly available Kubernetes clusters, where the Kubernetes and Kyma configurations are optimized for production use cases. The Kubernetes worker nodes are deployed in three availability zones of the respective [cloud region](https://help.sap.com/docs/btp/sap-business-technology-platform/regions-for-kyma-environment?locale=en-US&ai=true), and thus can provide zone level failure tolerance for Kyma and applications deployed on Kyma runtime. The [Kubernetes control plane](https://kubernetes.io/docs/reference/glossary/?all=true#term-control-plane) is also hosted in three availability zones of the respective region.

### Custom Worker Node Pools

You can configure additional worker node pools to suit your specific needs. We recommend enabling high availability in production environments to enhance resilience. Note that you cannot choose specific availability zones, as they are automatically selected by the platform during provisioning to ensure optimized deployment.

If you disable high availability, your additional worker node pool is deployed in a single availability zone, lacking zone-level failure tolerance.

> [!WARNING]
> You cannot change the high availability setting of an existing additional worker node pool. To alter this setting, you must delete and recreate the pool with the desired configuration.

High availability for additional worker node pools may depend on your choice of virtual machine types:

- For general-purpose machines, three availability zones are always available in all supported regions.
- For compute-intensive machines, the number of availability zones varies by region. For more information on machine types and regional availability, see [Machine Type: Machine Type in Additional Worker Node Pools](https://help.sap.com/docs/btp/sap-business-technology-platform/provisioning-and-update-parameters-in-kyma-environment?locale=en-US&version=Cloud#machine-type).

## User Application Deployment

While high availability is guaranteed for Kubernetes and native Kyma components, it is not automatically provided for your own applications deployed on Kyma. To ensure your applications are resilient to zone failures, you must manually configure high availability by following these guidelines:

- Deploy multiple replicas across different availability zones.
- Use appropriate resource requests and limits.
- Configure health checks and readiness probes.
- Implement proper service discovery and load balancing.

Deploying multiple replicas allows the Kubernetes scheduler to distribute them across zones, ensuring that if one zone becomes unavailable, your applications continue to run on replicas in the remaining zones.

For detailed guidance on building resilient applications, see [Develop Resilient Applications in the Kyma Runtime](https://help.sap.com/docs/btp/sap-business-technology-platform/resilient-applications-in-kyma-environment?locale=en-US).
