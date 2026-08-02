const picomatch = require('../index');

module.exports = function(input, options = {}) {
  return picomatch.scan(input, options);
};
