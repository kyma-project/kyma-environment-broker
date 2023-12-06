const {ensureOperationSucceeded} = require('../kyma-environment-broker/helpers');

async function ensureSuspensionSucceeded(keb, kcp, instanceID, operationID, timeout) {
  try {
    await ensureOperationSucceeded(keb, kcp, instanceID, operationID, timeout);
  } catch (e) {
    throw new Error(`Suspension failed: ${e.toString()}`);
  } finally {
    const runtimeStatus = await kcp.getRuntimeStatusOperations(instanceID);
    const events = await kcp.getRuntimeEvents(options.instanceID);
    console.log(`\nRuntime status after de-provisioning: ${runtimeStatus}\nEvents:\n${events}`);
    await kcp.reconcileInformationLog(runtimeStatus);
  }
}

async function callFuncAndPrintElapsedTime(fn) {
  const startTime = Date.now();
  const result = await fn();
  const endTime = Date.now();

  console.log(`Elapsed time: ${String(endTime - startTime)} ms`);
  return result;
}

module.exports = {
  ensureSuspensionSucceeded,
  callFuncAndPrintElapsedTime,
};
