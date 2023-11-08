# Kyma Environment Broker CronJobs

Kyma Environment Broker (KEB) comes with a set of CronJobs designed to clean up the environment that KEB runs on. They are not required for KEB to work, but their functions will not be performed if you don’t turn them on. 

| **Subcomponent** | **Description**|
| :--- | :--- |
|[Environments Cleanup CronJob](06-20-environments-cleanup-cronjob.md) | Cleans up environments that do not meet requirements in a given Gardener project.|
|[Subaccount Cleanup CronJob](06-30-subaccount-cleanup-cronjob.md) | Periodically calls the CIS service and notifies about SUBACCOUNT_DELETE events; based on these events, triggers the deprovisioning action on the Kyma runtime instance to which a given subaccount belongs. |
|[Trial Cleanup CronJob](06-40-trial-cleanup-cronjob.md) | Causes Kyma runtime instances with the trial plan to expire 14 days after their creation. |
|[Deprovision Retrigger CronJob](06-50-deprovision-retrigger-cronjob.md) | Makes another attempt to deprovision an instance. |
