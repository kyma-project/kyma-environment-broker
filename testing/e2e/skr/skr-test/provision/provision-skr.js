const {
  debug,
  kcp,
  gardener,
  keb,
  initK8sConfig,
  getSKRRuntimeStatus,
  initializeK8sClient,
} = require('../helpers');
const {expect} = require('chai');
const {provisionSKR, getShoot}= require('../../kyma-environment-broker');
const {BTPOperatorCreds} = require('../../smctl/helpers');
const {getSecret} = require('../../utils');

async function provisionSKRAndInitK8sConfig(options, provisioningTimeout) {
  console.log('Provisioning new SKR instance...');
  await provisionSKRInstance(options, provisioningTimeout);

  debug('Fetching runtime operation status...');
  const runtimeStatus = await kcp.getRuntimeStatusOperations(options.instanceID);
  const objRuntimeStatus = JSON.parse(runtimeStatus);
  expect(objRuntimeStatus).to.have.nested.property('data[0].shootName').not.empty;
  let shoot;
  if (process.env['GARDENER_KUBECONFIG']) {
    debug('Fetching shoot info from gardener...');
    shoot = await gardener.getShoot(objRuntimeStatus.data[0].shootName);
    debug(`Compass ID ${shoot.compassID}`);
  } else {
    debug('Fetching shoot info using kcp cli...');
    shoot = await getShoot(kcp, objRuntimeStatus.data[0].shootName);
  }

  if (process.env['GARDENER_KUBECONFIG']) {
    console.log('Initiating K8s config...');
    await initK8sConfig(shoot);
  } else {
    console.log('Initiating K8s client...');
    await initializeK8sClient({kubeconfigPath: shoot.kubeconfig});

    let retryCount = 0;
    const maxRetries = 10;
    const cooldown = 1000 * 60 * 1; // 1m

    while (retryCount < maxRetries) {
      try {
        await getSecret('sap-btp-manager', 'kyma-system');
        break;
      } catch (error) {
        console.log('An error occurred while testing the K8s client');
        console.log(`Downloading the kubeconfig again. Trying to initialize the client. Retry count: ${retryCount}`);
        const kubeconfigPath = await kcp.getKubeconfig(shoot.name);
        await initializeK8sClient({kubeconfigPath: kubeconfigPath});
        retryCount++;
        await new Promise((resolve) => setTimeout(resolve, cooldown));
      }
    }
  }
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

    await provisionSKR(keb,
        kcp,
        options.instanceID,
        options.runtimeName,
        null,
        btpOperatorCreds,
        options.customParams,
        timeout);

    debug('SKR is provisioned!');
  } catch (e) {
    throw new Error(`Provisioning failed: ${e.toString(), e.stack}`);
  } finally {
    debug('Fetching runtime status...');
    const runtimeStatus = await kcp.getRuntimeStatusOperations(options.instanceID);
    const events = await kcp.getRuntimeEvents(options.instanceID);
    console.log(`\nRuntime status after provisioning: ${runtimeStatus}\nEvents:\n${events}`);
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
  provisionSKRInstance,
};

