const {
  gatherOptions,
} = require('../skr-test');
const {provisionSKRAndInitK8sConfig} = require('../skr-test/provision/provision-skr');
const {deprovisionAndUnregisterSKR} = require('../skr-test/provision/deprovision-skr');
const {withCustomParams} = require('../skr-test');
const {expect} = require('chai');
const {KEBClient, KEBConfig} = require('../kyma-environment-broker');
const axios = require('axios');
const keb = new KEBClient(KEBConfig.fromEnv());

const provisioningTimeout = 1000 * 60 * 30; // 30m
const deprovisioningTimeout = 1000 * 60 * 95; // 95m
let globalTimeout = 1000 * 60 * 30; // 20m
const slowTime = 5000;

describe('SKR AWS networking test', function() {
  globalTimeout += provisioningTimeout + deprovisioningTimeout;

  this.timeout(globalTimeout);
  this.slow(slowTime);

  let skr;

  const customParams = {
    'networking': {'nodes': '10.253.0.0/21'},
  };
  let options = gatherOptions(
      withCustomParams(customParams),
  );
  console.log('Using custom parameters: %o', customParams);

  it('Try networking params which overlap with restricted IP range', async function() {
    const customParams = {'networking': {'nodes': '10.242.0.0/22'}};
    const payload = keb.buildPayload('wrong-nodes', 'id01234876', null, null, customParams);
    const endpoint = `service_instances/id01234876`;
    const config = await keb.buildRequest(payload, endpoint, 'put');

    try {
      await axios.request(config);
      fail('KEB must return an error');
    } catch (err) {
      expect(err.response.status).equal(400);
      expect(err.response.data.description).to.include('overlap');
      console.log('Got response:');
      console.log(err.response.data);
    }
  });
  it('Try networking params with invalid IP range', async function() {
    const customParams = {'networking': {'nodes': '333.242.0.0/22'}};
    const payload = keb.buildPayload('wrong-nodes', 'id01234873', null, null, customParams);
    const endpoint = `service_instances/id01234876`;
    const config = await keb.buildRequest(payload, endpoint, 'put');

    try {
      await axios.request(config);
      fail('KEB must return an error');
    } catch (err) {
      console.log('Got response:');
      console.log(err.response.data);
      expect(err.response.status).equal(400);
    }
  });
  it('Perform provisioning', async function() {
    this.timeout(provisioningTimeout);
    skr = await provisionSKRAndInitK8sConfig(options, provisioningTimeout);
    options = skr.options;
  });

  after('Clean up the resources', async function() {
    this.timeout(deprovisioningTimeout);
    await deprovisionAndUnregisterSKR(options, deprovisioningTimeout, true);
  });
});
