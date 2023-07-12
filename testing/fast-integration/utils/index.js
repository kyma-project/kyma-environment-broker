function getEnvOrThrow(key) {
  if (!process.env[key]) {
    throw new Error(`Env ${key} not present`);
  }

  return process.env[key];
}

module.exports = {
  getEnvOrThrow,
};
