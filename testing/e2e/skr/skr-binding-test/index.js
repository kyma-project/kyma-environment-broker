const {expect} = require('chai');
const {gatherOptions} = require('../skr-test');
const {initializeK8sClient} = require('../utils/index.js');
const {getSecret, getKubeconfigValidityInSeconds} = require('../utils');
const {provisionSKRInstance} = require('../skr-test/provision/provision-skr');
const {deprovisionAndUnregisterSKR} = require('../skr-test/provision/deprovision-skr');
const {KEBClient, KEBConfig} = require('../kyma-environment-broker');
const keb = new KEBClient(KEBConfig.fromEnv());

const provisioningTimeout = 1000 * 60 * 30; // 30m
const deprovisioningTimeout = 1000 * 60 * 95; // 95m
let globalTimeout = 1000 * 60 * 70; // 70m
const slowTime = 5000;
const secretName = 'sap-btp-manager';
const ns = 'kyma-system';

describe('SKR Binding test', function() {
  globalTimeout += provisioningTimeout + deprovisioningTimeout;

  this.timeout(globalTimeout);
  this.slow(slowTime);

  const options = gatherOptions(); // with default values
  let kubeconfigFromBinding;
  let bindingID;

  before('Ensure SKR is provisioned', async function() {
    this.timeout(provisioningTimeout);
    await provisionSKRInstance(options, provisioningTimeout);
  });

  it('Should not allow creation of more than 10 SKR bindings', async function() {
    errorOccurred = false;
    count = 0;
    while (!errorOccurred && count < 13) {
      bindingID = Math.random().toString(36).substring(2, 18);
      try {
        await keb.createBinding(options.instanceID, bindingID, true);
      } catch (err) {
        if (err.response) {
          errorOccurred = true;
          expect(err.response.status).equal(400);
          expect(err.response.data.description).to.include('maximum number of bindings reached');
          console.log('Got response:');
          console.log(err.response.data);
        } else {
          throw err;
        }
      }
      count++;
    }

    if (count >= 13) {
      expect.fail('The call was expected to fail but it passed. Created more than 10 bindings');
    }
  });

  after('Cleanup the resources', async function() {
    this.timeout(deprovisioningTimeout);
    if (process.env['SKIP_DEPROVISIONING'] != 'true') {
      await deprovisionAndUnregisterSKR(options, deprovisioningTimeout, true);
    }
  });
});
