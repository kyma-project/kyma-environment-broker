# Assured Workloads

## Overview

SAP BTP, Kyma runtime instances provisioned in the Google Cloud `cf-sa30` subaccount region require the Assured Workloads Kingdom of Saudi Arabia (KSA) control package.

Kyma runtime supports the BTP `cf-sa30` Google Cloud subaccount region, which is called the KSA BTP subaccount region.
Kyma Control Plane manages `cf-sa30` Kyma runtimes in a separate Google Cloud hyperscaler account pool.

When the **PlatformRegion** is a KSA BTP subaccount region, the KEB services catalog handler exposes
`me-central2` (KSA, Dammam) as the only possible value for the **region** parameter.
