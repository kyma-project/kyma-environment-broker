const k8s = require('@kubernetes/client-node');
const {expect} = require('chai');
const yaml = require('js-yaml');
const forge = require('node-forge');

const kc = new k8s.KubeConfig();
let k8sDynamicApi;
let k8sAppsApi;
let k8sCoreV1Api;
let k8sRbacAuthorizationV1Api;

let watch;

function initializeK8sClient(opts) {
  opts = opts || {};

  k8sDynamicApi = null;
  k8sAppsApi = null;
  k8sCoreV1Api = null;
  k8sRbacAuthorizationV1Api = null;
  watch = null;
  k8sLog = null;
  k8sServerUrl = null;

  try {
    console.log('Trying to initialize a K8S client');
    if (opts.kubeconfigPath) {
      console.log('Path initialization');
      kc.loadFromFile(opts.kubeconfigPath);
    } else if (opts.kubeconfig) {
      console.log('Kubeconfig initialization');
      kc.loadFromString(opts.kubeconfig);
    } else {
      console.log('Default initialization');
      kc.loadFromDefault();
    }

    console.log('Clients creation');
    k8sDynamicApi = kc.makeApiClient(k8s.KubernetesObjectApi);
    console.log('Making Api client - Apps');
    k8sAppsApi = kc.makeApiClient(k8s.AppsV1Api);
    console.log('Making Api client - Core');
    k8sCoreV1Api = kc.makeApiClient(k8s.CoreV1Api);
    console.log('Making Api client - Auth');
    k8sRbacAuthorizationV1Api = kc.makeApiClient(k8s.RbacAuthorizationV1Api);
    console.log('Making Api client - Logs');
    k8sLog = new k8s.Log(kc);
    console.log('Making Api client - Watch');
    watch = new k8s.Watch(kc);
    k8sServerUrl = kc.getCurrentCluster() ? kc.getCurrentCluster().server : null;
  } catch (err) {
    console.log(err.message);
  }
}
initializeK8sClient();

function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

async function k8sDelete(listOfSpecs, namespace) {
  for (const res of listOfSpecs) {
    if (namespace) {
      res.metadata.namespace = namespace;
    }
    debug(`Delete ${res.metadata.name}`);
    try {
      if (res.kind) {
        await k8sDynamicApi.delete(res);
      } else if (res.metadata.selfLink) {
        await k8sDynamicApi.requestPromise({
          url: k8sDynamicApi.basePath + res.metadata.selfLink,
          method: 'DELETE',
        });
      } else {
        throw Error(
            'Object kind or metadata.selfLink is required to delete the resource',
        );
      }
      if (res.kind === 'CustomResourceDefinition') {
        const version = res.spec.version || res.spec.versions[0].name;
        const path = `/apis/${res.spec.group}/${version}/${res.spec.names.plural}`;
        await deleteAllK8sResources(path);
      }
    } catch (err) {
      ignore404(err);
    }
  }
}

async function getSecret(name, namespace) {
  const path = `/api/v1/namespaces/${namespace}/secrets/${name}`;
  const response = await k8sDynamicApi.requestPromise({
    url: k8sDynamicApi.basePath + path,
  });
  return JSON.parse(response.body);
}

async function k8sApply(resources, namespace, patch = true) {
  const options = {
    headers: {'Content-type': 'application/merge-patch+json'},
  };
  for (const resource of resources) {
    if (!resource || !resource.kind || !resource.metadata.name) {
      debug('Skipping invalid resource:', resource);
      continue;
    }
    if (!resource.metadata.namespace) {
      resource.metadata.namespace = namespace;
    }
    if (resource.kind == 'Namespace') {
      resource.metadata.labels = {
        'istio-injection': 'enabled',
      };
    }
    try {
      await k8sDynamicApi.patch(
          resource,
          undefined,
          undefined,
          undefined,
          undefined,
          options,
      );
      debug(resource.kind, resource.metadata.name, 'reconfigured');
    } catch (e) {
      {
        if (e.body && e.body.reason === 'NotFound') {
          try {
            await k8sDynamicApi.create(resource);
            debug(resource.kind, resource.metadata.name, 'created');
          } catch (createError) {
            debug(resource.kind, resource.metadata.name, 'failed to create');
            debug(JSON.stringify(createError, null, 4));
            throw createError;
          }
        } else {
          throw e;
        }
      }
    }
  }
}

// Allows to pass watch with different than global K8S context.
function waitForK8sObject(path, query, checkFn, timeout, timeoutMsg, watcher = watch) {
  debug('waiting for', path);
  let res;
  let timer;
  return new Promise((resolve, reject) => {
    watcher.watch(
        path,
        query,
        (type, apiObj, watchObj) => {
          if (checkFn(type, apiObj, watchObj)) {
            if (res) {
              res.abort();
            }
            clearTimeout(timer);
            debug('finished waiting for ', path);
            resolve(watchObj.object);
          }
        },
        () => {
        },
    )
        .then((r) => {
          res = r;
          timer = setTimeout(() => {
            res.abort();
            reject(new Error(timeoutMsg));
          }, timeout);
        });
  });
}

function waitForSecret(
    secretName,
    namespace = 'default',
    timeout = 90_000,
) {
  return waitForK8sObject(
      `/api/v1/namespaces/${namespace}/secrets`,
      {},
      (_type, _apiObj, watchObj) => {
        return watchObj.object.metadata.name.includes(
            secretName,
        );
      },
      timeout,
      `Waiting for ${secretName} Secret timeout (${timeout} ms)`,
  );
}

async function getSecretData(name, namespace) {
  try {
    const secret = await getSecret(name, namespace);
    const encodedData = secret.data;
    return Object.fromEntries(
        Object.entries(encodedData).map(([key, value]) => {
          const buff = Buffer.from(value, 'base64');
          const decoded = buff.toString('ascii');
          return [key, decoded];
        }),
    );
  } catch (e) {
    console.log('Error:', e);
    throw e;
  }
}

function ignore404(e) {
  if (
    (e.statusCode && e.statusCode === 404) ||
      (e.response && e.response.statusCode && e.response.statusCode === 404)
  ) {
    debug('Warning: Ignoring NotFound error');
    return;
  }

  throw e;
}

// NOTE: this no longer works, it relies on kube-api sending `selfLink` but the field has been deprecated
async function deleteAllK8sResources(
    path,
    query = {},
    retries = 2,
    interval = 1000,
    keepFinalizer = false,
) {
  try {
    let i = 0;
    while (i < retries) {
      if (i++) {
        await sleep(interval);
      }
      const response = await k8sDynamicApi.requestPromise({
        url: k8sDynamicApi.basePath + path,
        qs: query,
      });
      const body = JSON.parse(response.body);
      if (body.items && body.items.length) {
        for (const o of body.items) {
          await deleteK8sResource(o, path, keepFinalizer);
        }
      } else if (!body.items) {
        await deleteK8sResource(body, path, keepFinalizer);
      }
    }
  } catch (e) {
    debug('Error during delete ', path, String(e).substring(0, 1000));
    debug(e);
  }
}

async function deleteK8sResource(o, path, keepFinalizer = false) {
  if (o.metadata.finalizers && o.metadata.finalizers.length && !keepFinalizer) {
    const options = {
      headers: {'Content-type': 'application/merge-patch+json'},
    };

    const obj = {
      kind: o.kind || 'Secret', // Secret list doesn't return kind and apiVersion
      apiVersion: o.apiVersion || 'v1',
      metadata: {
        name: o.metadata.name,
        namespace: o.metadata.namespace,
        finalizers: [],
      },
    };

    debug('Removing finalizers from', obj);
    try {
      await k8sDynamicApi.patch(obj, undefined, undefined, undefined, undefined, options);
    } catch (err) {
      ignore404(err);
    }
  }

  try {
    let objectUrl = `${k8sDynamicApi.basePath + path}/${o.metadata.name}`;
    if (o.metadata.selfLink) {
      debug('using selfLink for deleting object');
      objectUrl = k8sDynamicApi.basePath + o.metadata.selfLink;
    }

    debug('Deleting resource: ', objectUrl);
    await k8sDynamicApi.requestPromise({
      url: objectUrl,
      method: 'DELETE',
    });
  } catch (err) {
    ignore404(err);
  }

  debug(
      'Deleted resource:',
      o.metadata.name,
      'namespace:',
      o.metadata.namespace,
  );
}

async function getKymaAdminBindings() {
  const {body} = await k8sRbacAuthorizationV1Api.listClusterRoleBinding();
  const adminRoleBindings = body.items;
  return adminRoleBindings
      .filter(
          (clusterRoleBinding) => clusterRoleBinding.roleRef.name === 'cluster-admin',
      )
      .map((clusterRoleBinding) => ({
        name: clusterRoleBinding.metadata.name,
        role: clusterRoleBinding.roleRef.name,
        users: clusterRoleBinding.subjects
            .filter((sub) => sub.kind === 'User')
            .map((sub) => sub.name),
        groups: clusterRoleBinding.subjects
            .filter((sub) => sub.kind === 'Group')
            .map((sub) => sub.name),
      }));
}

async function findKymaAdminBindingForUser(targetUser) {
  const kymaAdminBindings = await getKymaAdminBindings();
  return kymaAdminBindings.find(
      (binding) => binding.users.indexOf(targetUser) >= 0,
  );
}

async function ensureKymaAdminBindingExistsForUser(targetUser) {
  const binding = await findKymaAdminBindingForUser(targetUser);
  expect(binding).not.to.be.undefined;
  expect(binding.users).to.include(targetUser);
}

async function ensureKymaAdminBindingDoesNotExistsForUser(targetUser) {
  const binding = await findKymaAdminBindingForUser(targetUser);
  expect(binding).to.be.undefined;
}

let DEBUG = process.env.DEBUG === 'true';

function log(prefix, ...args) {
  if (args.length === 0) {
    return;
  }

  args = [...args];
  const fmt = `[${prefix}] ` + args[0];
  args = args.slice(1);
  console.log.apply(console, [fmt, ...args]);
}

function isDebugEnabled() {
  return DEBUG;
}

function switchDebug(on = true) {
  DEBUG = on;
}

function debug(...args) {
  if (!isDebugEnabled()) {
    return;
  }
  log('DEBUG', ...args);
}

function info(...args) {
  log('INFO', ...args);
}

function error(...args) {
  log('ERROR', ...args);
}

function fromBase64(s) {
  return Buffer.from(s, 'base64').toString('utf8');
}

function toBase64(s) {
  return Buffer.from(s).toString('base64');
}

function genRandom(len) {
  let res = '';
  const chrs = 'abcdefghijklmnopqrstuvwxyz0123456789';
  for (let i = 0; i < len; i++) {
    res += chrs.charAt(Math.floor(Math.random() * chrs.length));
  }

  return res;
}

function getEnvOrThrow(key) {
  if (!process.env[key]) {
    throw new Error(`Env ${key} not present`);
  }

  return process.env[key];
}

function wait(fn, checkFn, timeout, interval) {
  return new Promise((resolve, reject) => {
    const th = setTimeout(function() {
      debug('wait timeout');
      done(reject, new Error('wait timeout'));
    }, timeout);
    const ih = setInterval(async function() {
      let res;
      try {
        res = await fn();
      } catch (ex) {
        res = ex;
      }
      checkFn(res) && done(resolve, res);
    }, interval);

    function done(fn, arg) {
      clearTimeout(th);
      clearInterval(ih);
      fn(arg);
    }
  });
}

function getKubeconfigValidityInSeconds(kubeconfig) {
  try {
    const doc = yaml.load(kubeconfig);
    const users = doc.users;
    if (users && users.length > 0) {
      const pem = users[0].user['client-certificate-data'];
      const decodedPem = atob(pem);
      const certificate = forge.pki.certificateFromPem(decodedPem);
      const difference = certificate.validity.notAfter.getTime() - certificate.validity.notBefore.getTime();
      return difference / 1000;
    } else {
      console.error('No user data found');
      return null;
    }
  } catch (e) {
    console.error('Error parsing YAML content:', e);
    return null;
  }
}

module.exports = {
  initializeK8sClient,
  k8sApply,
  k8sDelete,
  waitForK8sObject,
  waitForSecret,
  ensureKymaAdminBindingExistsForUser,
  ensureKymaAdminBindingDoesNotExistsForUser,
  getSecret,
  getSecretData,
  k8sDynamicApi,
  k8sAppsApi,
  k8sCoreV1Api,
  info,
  error,
  debug,
  switchDebug,
  isDebugEnabled,
  fromBase64,
  toBase64,
  genRandom,
  getEnvOrThrow,
  wait,
  getKubeconfigValidityInSeconds,
};
