const {KCPWrapper, KCPConfig} = require('../kcp/client');
const {KEBClient, KEBConfig} = require('../kyma-environment-broker');
const {gatherOptions} = require('../skr-test/helpers');
const {provisionSKRInstance} = require('../skr-test/provision/provision-skr');
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
  const options = gatherOptions();

  before('Ensure SKR Trial is provisioned', async function() {
    try {
      await callFuncAndPrintExecutionTime(provisionSKRInstance, [options, provisioningTimeout]);
    } catch (e) {
      throw new Error(`${e.toString()}\n`);
    }
  });

  it('should wait until Trial Cleanup CronJob triggers suspension', async function() {
    try {
      const rs = await callFuncAndPrintExecutionTime(ensureSuspensionIsInProgress,
          [kcp, options.instanceID, trialCleanupTriggerTimeout]);
      suspensionOpID = rs.data[0].status[suspensionOperationType].data[0].operationID;
      assert.isDefined(suspensionOpID, `suspension operation ID: ${suspensionOpID}`);
    } catch (e) {
      throw new Error(`${e.toString()}\n`);
    }
  });

  it('should wait until suspension succeeds', async function() {
    try {
      debug(`Waiting until suspension operation succeeds...`);
      await callFuncAndPrintExecutionTime(ensureSuspensionSucceeded,
          [keb, kcp, options.instanceID, suspensionOpID, suspensionTimeout]);
    } catch (e) {
      throw new Error(`${e.toString()}\n`);
    }
  });

  after('Cleanup the resources', async function() {
    await callFuncAndPrintExecutionTime(deprovisionAndUnregisterSKR,
        [options, deprovisioningAfterSuspensionTimeout, true]);
  });
});
