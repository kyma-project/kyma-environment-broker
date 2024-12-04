const {wait, debug} = require('../utils');
const {expect} = require('chai');
const fs = require('fs');
const os = require('os');

async function provisionSKR(
    keb,
    kcp,
    instanceID,
    name,
    platformCreds,
    btpOperatorCreds,
    customParams,
    timeout,
) {
  const resp = await keb.provisionSKR(
      name,
      instanceID,
      platformCreds,
      btpOperatorCreds,
      customParams,
  );
  expect(resp).to.have.property('operation');

  const operationID = resp.operation;
  debug(`Operation ID ${operationID}`);

  await ensureOperationSucceeded(keb, kcp, instanceID, operationID, timeout);
}

async function getShoot(kcp, shootName) {
  debug(`Fetching shoot: ${shootName}`);

  const kubeconfigPath = await kcp.getKubeconfig(shootName);

  // try to get data from Provisioner (GardenerConfig) and Runtime resource
  const runtimeGardenerConfig = await kcp.getRuntimeGardenerConfig(shootName);
  const objRuntimeGardenerConfig = JSON.parse(runtimeGardenerConfig);
  // use Gardener config or Runtime Config based on the presence of gardenerConfig field
  if (objRuntimeGardenerConfig.data[0].status.gardenerConfig != null) {
    expect(objRuntimeGardenerConfig).to.have.nested.property('data[0].status.gardenerConfig.oidcConfig').not.empty;
    expect(objRuntimeGardenerConfig).to.have.nested.property('data[0].status.gardenerConfig.machineType').not.empty;
    return {
      name: shootName,
      kubeconfig: kubeconfigPath,
      oidcConfig: objRuntimeGardenerConfig.data[0].status.gardenerConfig.oidcConfig,
      machineType: objRuntimeGardenerConfig.data[0].status.gardenerConfig.machineType,
    };
  } else {
    expect(objRuntimeGardenerConfig).to.have.nested.
        property('data[0].runtimeConfig.spec.shoot.kubernetes.kubeAPIServer.oidcConfig').not.empty;
    expect(objRuntimeGardenerConfig).to.have.nested.
        property('data[0].runtimeConfig.spec.shoot.provider.workers[0].machine.type').not.empty;
    return {
      name: shootName,
      kubeconfig: kubeconfigPath,
      oidcConfig: objRuntimeGardenerConfig.data[0].runtimeConfig.spec.shoot.kubernetes.kubeAPIServer.oidcConfig,
      machineType: objRuntimeGardenerConfig.data[0].runtimeConfig.spec.shoot.provider.workers[0].machine.type,
    };
  }
}

function ensureValidShootOIDCConfig(shoot, targetOIDCConfig) {
  expect(shoot).to.have.nested.property('oidcConfig.clientID', targetOIDCConfig.clientID);
  expect(shoot).to.have.nested.property('oidcConfig.issuerURL', targetOIDCConfig.issuerURL);
  expect(shoot).to.have.nested.property('oidcConfig.groupsClaim', targetOIDCConfig.groupsClaim);
  expect(shoot).to.have.nested.property('oidcConfig.usernameClaim', targetOIDCConfig.usernameClaim);
  expect(shoot).to.have.nested.property(
      'oidcConfig.usernamePrefix',
      targetOIDCConfig.usernamePrefix,
  );
  expect(shoot.oidcConfig.signingAlgs).to.eql(targetOIDCConfig.signingAlgs);
}

async function deprovisionSKR(keb, kcp, instanceID, timeout, ensureSuccess=true) {
  const resp = await keb.deprovisionSKR(instanceID);
  expect(resp).to.have.property('operation');

  const operationID = resp.operation;
  console.log(`Deprovision SKR - operation ID ${operationID}`);

  if (ensureSuccess) {
    await ensureOperationSucceeded(keb, kcp, instanceID, operationID, timeout);
  }

  return operationID;
}

async function updateSKR(keb,
    kcp,
    gardener,
    instanceID,
    shootName,
    customParams,
    timeout,
    btpOperatorCreds = null,
    isMigration = false) {
  const resp = await keb.updateSKR(instanceID, customParams, btpOperatorCreds, isMigration);
  expect(resp).to.have.property('operation');

  const operationID = resp.operation;
  debug(`Operation ID ${operationID}`);

  await ensureOperationSucceeded(keb, kcp, instanceID, operationID, timeout);

  let shoot;

  if (process.env['GARDENER_KUBECONFIG']) {
    shoot = await gardener.getShoot(shootName);
  } else {
    shoot = await getShoot(kcp, shootName);
  }

  return {
    operationID,
    shoot,
  };
}

async function ensureOperationSucceeded(keb, kcp, instanceID, operationID, timeout) {
  const res = await wait(
      () => keb.getOperation(instanceID, operationID),
      (res) => res && res.state && (res.state === 'succeeded' || res.state === 'failed'),
      timeout,
      1000 * 30, // 30 seconds
  ).catch(async (err) => {
    const runtimeStatus = await kcp.getRuntimeStatusOperations(instanceID);
    const events = await kcp.getRuntimeEvents(instanceID);
    const msg = `${err}\nError thrown by ensureOperationSucceeded: Runtime status: ${runtimeStatus}`;
    throw new Error(`${msg}\nEvents:\n${events}`);
  });

  if (res.state !== 'succeeded') {
    const runtimeStatus = await kcp.getRuntimeStatusOperations(instanceID);
    throw new Error(`Error thrown by ensureOperationSucceeded: operation didn't succeed in time:
     ${JSON.stringify(res, null, `\t`)}\nRuntime status: ${runtimeStatus}`);
  }

  console.log(`Operation ${operationID} finished with state ${res.state}`);

  return res;
}

async function getShootName(keb, instanceID) {
  const resp = await keb.getRuntime(instanceID);
  expect(resp.data).to.be.lengthOf(1);

  return resp.data[0].shootName;
}

async function getCatalog(keb) {
  return keb.getCatalog();
}

async function ensureValidOIDCConfigInCustomerFacingKubeconfig(keb, instanceID, oidcConfig) {
  let kubeconfigContent;
  try {
    kubeconfigContent = await keb.downloadKubeconfig(instanceID);
  } catch (err) {
    console.log(err);
  }

  const issuerMatchPattern = '\\b' + oidcConfig.issuerURL + '\\b';
  const clientIDMatchPattern = '\\b' + oidcConfig.clientID + '\\b';
  expect(kubeconfigContent).to.match(new RegExp(issuerMatchPattern, 'g'));
  expect(kubeconfigContent).to.match(new RegExp(clientIDMatchPattern, 'g'));
}

async function saveKubeconfig(kubeconfig) {
  const directory = `${os.homedir()}/.kube`;
  if (!fs.existsSync(directory)) {
    fs.mkdirSync(directory, {recursive: true});
  }

  fs.writeFileSync(`${directory}/config`, kubeconfig);
}

module.exports = {
  provisionSKR,
  deprovisionSKR,
  saveKubeconfig,
  updateSKR,
  ensureOperationSucceeded,
  getShootName,
  ensureValidShootOIDCConfig,
  ensureValidOIDCConfigInCustomerFacingKubeconfig,
  getCatalog,
  getShoot,
};
