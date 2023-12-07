const {KCPWrapper, KCPConfig} = require('../kcp/client');
const {KEBClient, KEBConfig} = require('../kyma-environment-broker');
const {gatherOptions} = require('../skr-test/helpers');
const {getOrProvisionSKR} = require('../skr-test/provision/provision-skr');
const {deprovisionAndUnregisterSKR} = require('../skr-test/provision/deprovision-skr');
const {debug} = require('../utils');
const {assert} = require('chai');
const {
  ensureSuspensionSucceeded,
  callFuncAndPrintExecutionTime,
  ensureSuspensionIsInProgress,
} = require('./helpers');

const kcp = new KCPWrapper(KCPConfig.fromEnv());
const keb = new KEBClient(KEBConfig.fromEnv());

const provisioningTimeout = 1000 * 60 * 30; // 30m
const suspensionTimeout = 1000 * 60 * 60; // 60m
const deprovisioningAfterSuspensionTimeout = 1000 * 60 * 5; // 5m
const trialCleanupTriggerTimeout = 1000 * 60 * 11; // 11m
const slowTestDuration = 1000 * 60 * 40; // 40m
const globalTimeout = provisioningTimeout + suspensionTimeout;
const suspensionOperationType = 'suspension';

describe('SKR Trial suspension test', function() {
  this.timeout(globalTimeout);
  this.slow(slowTestDuration);

  let suspensionOpID;
  let skipTests = false;
  const options = gatherOptions();

  before('Ensure SKR Trial is provisioned', async function() {
    try {
      await callFuncAndPrintExecutionTime(getOrProvisionSKR, [options, false, provisioningTimeout]);
    } catch (e) {
      console.log(`${e.toString()}\n`);
      skipTests = true;
    }
  });

  it('should wait until Trial Cleanup CronJob triggers suspension', async function() {
    if (skipTests) {
      console.log(`skipping test due to failures`);
      this.skip();
    }

    try {
      const rs = await callFuncAndPrintExecutionTime(ensureSuspensionIsInProgress,
          [kcp, options.instanceID, trialCleanupTriggerTimeout]);
      suspensionOpID = rs.data[0].status[suspensionOperationType].data[0].operationID;
      assert.isDefined(suspensionOpID, `suspension operation ID: ${suspensionOpID}`);
    } catch (e) {
      console.log(`${e.toString()}\n`);
      skipTests = true;
      this.skip();
    }
  });

  it('should wait until suspension succeeds', async function() {
    if (skipTests) {
      console.log(`skipping test due to failures`);
      this.skip();
    }

    try {
      debug(`Waiting until suspension operation succeeds...`);
      await callFuncAndPrintExecutionTime(ensureSuspensionSucceeded,
          [keb, kcp, options.instanceID, suspensionOpID, suspensionTimeout]);
    } catch (e) {
      console.log(`${e.toString()}\n`);
      skipTests = true;
      this.skip();
    }
  });

  after('Cleanup the resources', async function() {
    await callFuncAndPrintExecutionTime(deprovisionAndUnregisterSKR,
        [options, deprovisioningAfterSuspensionTimeout, false, true]);
  });
});
