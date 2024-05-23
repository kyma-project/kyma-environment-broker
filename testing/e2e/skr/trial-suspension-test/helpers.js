const {ensureOperationSucceeded} = require('../kyma-environment-broker/helpers');

const suspensionOperationType = 'suspension';
const inProgressOperationState = 'in progress';

async function callFuncAndPrintExecutionTime(callbackFn, callbackFnArgs) {
  const startTime = Date.now();
  try {
    const result = await callbackFn.apply(this, callbackFnArgs);
    return result;
  } catch (e) {
    throw new Error(`callFuncAndPrintExecutionTime: failed during "${callbackFn.name}": ${e.toString()}\n`);
  } finally {
    const endTime = Date.now();
    console.log(`\n"${callbackFn.name}" execution time: ${String(endTime - startTime)} ms\n`);
  }
}

async function ensureSuspensionIsInProgress(kcp, instanceID, timeout) {
  try {
    const res = await kcp.ensureLatestGivenOperationTypeIsInGivenState(
        instanceID, suspensionOperationType, inProgressOperationState, timeout);
    return res;
  } catch (e) {
    console.log(`cannot ensure "${suspensionOperationType}" operation to be in state "${inProgressOperationState}"`);
    throw new Error(`ensureLatestGivenOperationTypeIsInGivenState failed: ${e.toString()}\n`);
  } finally {
    const runtimeStatus = await kcp.getRuntimeStatusOperations(instanceID);
    const events = await kcp.getRuntimeEvents(instanceID);
    console.log(`Runtime status:\n${runtimeStatus}\nEvents:\n${events}`);
  }
}

async function ensureSuspensionSucceeded(keb, kcp, instanceID, operationID, timeout) {
  try {
    await ensureOperationSucceeded(keb, kcp, instanceID, operationID, timeout);
  } catch (e) {
    console.log(`cannot ensure "${suspensionOperationType}" operation success: ${e.toString()}\n`);
    throw new Error(`ensureOperationSucceeded failed: ${e.toString()}\n`);
  } finally {
    const runtimeStatus = await kcp.getRuntimeStatusOperations(instanceID);
    const events = await kcp.getRuntimeEvents(instanceID);
    console.log(`Runtime status after suspension:\n${runtimeStatus}\nEvents:\n${events}`);
  }
}

module.exports = {
  callFuncAndPrintExecutionTime,
  ensureSuspensionSucceeded,
  ensureSuspensionIsInProgress,
};
