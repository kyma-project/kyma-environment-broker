# The Additional Worker Node Pool Autoscaler

Additional Worker Node Pools specified in the `additionalWorkerNodePools` provisioning/update parameter require **autoScalerMin** and **autoScalerMax** values to configure the autoscaler for the selected additional worker node pool. 
See the [Additional Worker Node Pools](../user/04-40-additional-worker-node-pools.md) for parameter details.

Setting the **autoScalerMin** value to 0 results in an automatic removal of the additional worker nodes depending on the current workload in the cluster. If there are no user's workloads deployed onto the nodes associated with the given additional worker node pool, the nodes should be removed automatically in around 30 mins.