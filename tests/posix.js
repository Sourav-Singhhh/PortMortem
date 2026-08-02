const picomatch = require('./index');

module.exports = function(glob, options = {}) {
  return picomatch(glob, { ...options, posix: true });
};
