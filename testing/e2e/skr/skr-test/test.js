const {
  gatherOptions,
  oidcE2ETest,
  machineTypeE2ETest,
  // btpManagerSecretTest,
} = require('./index');
const {provisionSKRAndInitK8sConfig} = require('./provision/provision-skr');
const {deprovisionAndUnregisterSKR} = require('./provision/deprovision-skr');

const provisioningTimeout = 1000 * 60 * 30; // 30m
const deprovisioningTimeout = 1000 * 60 * 95; // 95m
let globalTimeout = 1000 * 60 * 70; // 70m
const slowTime = 5000;

describe('SKR test', function() {
  globalTimeout += provisioningTimeout + deprovisioningTimeout;

  this.timeout(globalTimeout);
  this.slow(slowTime);

  let options = gatherOptions(); // with default values
  let skr;
  const getShootInfoFunc = function() {
    return skr.shoot;
  };
  const getShootOptionsFunc = function() {
    return options;
  };

  before('Ensure SKR is provisioned', async function() {
    this.timeout(provisioningTimeout);
    skr = await provisionSKRAndInitK8sConfig(options, provisioningTimeout);
    options = skr.options;
  });

  // Run BTP Manager Secret tests
  // btpManagerSecretTest();

  // Run OIDC tests
  oidcE2ETest(getShootOptionsFunc, getShootInfoFunc);

  // Run Machine Type tests
  machineTypeE2ETest(getShootOptionsFunc, getShootInfoFunc);

  after('Cleanup the resources', async function() {
    this.timeout(deprovisioningTimeout);
    if (process.env['SKIP_DEPROVISIONING'] != 'true') {
      await deprovisionAndUnregisterSKR(options, deprovisioningTimeout, true);
    }
  });
});
