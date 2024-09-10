# Assured Workloads

## Overview

SKRs provisioned in the GCP `cf-sa30` subaccount region require Assured Workloads Kingdom of Saudi Arabia (KSA) control package.

SAP BTP, Kyma runtime supports the BTP `cf-sa30` GCP subaccount region, which is called KSA BTP subaccount region.
Kyma Control Plane manages `cf-sa30` Kyma runtimes in a separate GCP hyperscaler account pool.

When the **PlatformRegion** is a KSA BTP subaccount region, the KEB services catalog handler exposes
`me-central2` (KSA, Dammam) as the only possible value for the **region** parameter.
