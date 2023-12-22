const {
  debug,
  kcp,
  gardener,
  keb,
  initK8sConfig,
  getSKRRuntimeStatus,
} = require('../helpers');

const {provisionSKR}= require('../../kyma-environment-broker');
const {BTPOperatorCreds} = require('../../smctl/helpers');

async function provisionSKRAndInitK8sConfig(options, provisioningTimeout) {
  console.log('Provisioning new SKR instance...');
  const shoot = await provisionSKRInstance(options, provisioningTimeout);

  console.log('Initiating K8s config...');
  await initK8sConfig(shoot);
  console.log('Initialization of K8s finished...');

  return {
    options,
    shoot,
  };
}

async function provisionSKRInstance(options, timeout) {
  try {
    const btpOperatorCreds = BTPOperatorCreds.dummy();

    console.log(`\nInstanceID ${options.instanceID}`,
        `Runtime ${options.runtimeName}`, `Application ${options.appName}`, `Suffix ${options.suffix}`);

    const skr = await provisionSKR(keb,
        kcp, gardener,
        options.instanceID,
        options.runtimeName,
        null,
        btpOperatorCreds,
        options.customParams,
        timeout);

    debug('SKR is provisioned!');
    return skr.shoot;
  } catch (e) {
    throw new Error(`Provisioning failed: ${e.toString(), e.stack}`);
  } finally {
    debug('Fetching runtime status...');
    const runtimeStatus = await kcp.getRuntimeStatusOperations(options.instanceID);
    const events = await kcp.getRuntimeEvents(options.instanceID);
    console.log(`\nRuntime status after provisioning: ${runtimeStatus}\nEvents:\n${events}`);
    await kcp.reconcileInformationLog(runtimeStatus);
  }
}

async function getSKRKymaVersion(instanceID) {
  const runtimeStatus = await getSKRRuntimeStatus(instanceID);
  if (runtimeStatus && runtimeStatus.data) {
    return runtimeStatus.data[0].kymaVersion;
  }
  return '';
}

module.exports = {
  provisionSKRAndInitK8sConfig,
  getSKRKymaVersion,
};

