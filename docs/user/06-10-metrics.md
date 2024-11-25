# Kyma Environment Broker Metrics

Kyma Environment Broker (KEB) metrics are exposed with the `/metrics` endpoint, which is called by Prometheus in a job manner every given duration of
time and gathers the metrics, which are later pushed to Victoria metrics and Plutno dashboard and alerts.

The metrics are counted in KEB's memory in two ways:

* Pulling data from the database and updating in-memory metrics.
* Publishing events sent across systems from business processes to corresponding subscribers, which update metrics in memory.

Then, the Prometheus server pulls the metrics stored in KEB's memory and persist them in monitoring database for further processing by monitoring and alert systems.

![KEB metrics](../assets/metrics.drawio.svg)

| Metric Name                                            | Type      | Labels                                                                                                  | Source            |
|--------------------------------------------------------|-----------|---------------------------------------------------------------------------------------------------------|-------------------|
| kcp_keb_v2_ers_context_license_type_total              | gauge     | license_type                                                                                            | database          |
| kcp_keb_v2_global_account_id_instances_total           | gauge     | global_account_id                                                                                       | database          |
| kcp_keb_v2_instances_total                             | gauge     | -                                                                                                       | database          |
| kcp_keb_v2_deprovisioning_duration_minutes             | histogram | plan_id                                                                                                 | event             |
| kcp_keb_v2_provisioning_duration_minutes               | histogram | plan_id                                                                                                 | event             |
| kcp_keb_v2_operation_result                            | gauge     | operation_id, instance_id, global_account_id, plan_id, type, state, error_category, error_reason, error | event             |
| kcp_keb_v2_operations_provisioning_failed_total        | counter   | plan_id                                                                                                 | event + database  |
| kcp_keb_v2_operations_provisioning_in_progress_total   | gauge     | plan_id                                                                                                 | event + database  |
| kcp_keb_v2_operations_provisioning_succeeded_total     | counter   | plan_id                                                                                                 | event + database  |
| kcp_keb_v2_operations_deprovisioning_failed_total      | counter   | plan_id                                                                                                 | event + database  |
| kcp_keb_v2_operations_deprovisioning_in_progress_total | gauge     | plan_id                                                                                                 | event + database  |
| kcp_keb_v2_operations_deprovisioning_succeeded_total   | counter   | plan_id                                                                                                 | event + database  |
| kcp_keb_v2_operations_update_failed_total              | counter   | plan_id                                                                                                 | event + database  |
| kcp_keb_v2_operations_update_in_progress_total         | gauge     | plan_id                                                                                                 | event + database  |
| kcp_keb_v2_operations_update_succeeded_total           | counter   | plan_id                                                                                                 | event + database  |
