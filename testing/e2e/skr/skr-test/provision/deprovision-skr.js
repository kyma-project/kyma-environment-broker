const {deprovisionSKR} = require('../../kyma-environment-broker');
const {keb, kcp} = require('../helpers');

async function deprovisionAndUnregisterSKR(options, deprovisioningTimeout, ensureSuccess) {
  await deprovisionSKRInstance(options, deprovisioningTimeout, ensureSuccess);
}

async function deprovisionSKRInstance(options, timeout, ensureSuccess=true) {
  try {
    await deprovisionSKR(keb, kcp, options.instanceID, timeout, ensureSuccess);
  } catch (e) {
    throw new Error(`De-provisioning failed: ${e.toString()}`);
  } finally {
    const runtimeStatus = await kcp.getRuntimeStatusOperations(options.instanceID);
    const events = await kcp.getRuntimeEvents(options.instanceID);
    console.log(`\nRuntime status after de-provisioning: ${runtimeStatus}\nEvents:\n${events}`);
  }
}

module.exports = {
  deprovisionAndUnregisterSKR,
};
