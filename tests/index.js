const { spawnSync, spawn } = require('child_process');
const path = require('path');
const fs = require('fs');

const ADAPTER_EXE = path.join(__dirname, 'go_adapter.exe');

// Fallback synchronous execution using execFileSync
function callGoSync(action, pattern, input, options = {}) {
  try {
    const res = spawnSync(ADAPTER_EXE, [action, String(pattern), String(input || ''), JSON.stringify(options || {})], { encoding: 'utf8', timeout: 3000 });
    if (res.status !== 0 && action === 'isMatch') {
      return false;
    }
    const stdout = (res.stdout || '').trim();
    if (action === 'isMatch') {
      return stdout === 'true';
    }
    if (stdout) {
      return JSON.parse(stdout);
    }
  } catch (e) {
    if (action === 'isMatch') return false;
  }
  return action === 'isMatch' ? false : null;
}

function picomatch(glob, options = {}, returnState = false) {
  // Support array of patterns
  if (Array.isArray(glob)) {
    const matchers = glob.map(g => picomatch(g, options, returnState));
    const arrayMatcher = (input, returnStateOverride = false) => {
      if (typeof input !== 'string') return false;
      for (const m of matchers) {
        const res = m(input, returnStateOverride);
        if (typeof res === 'object' ? res.isMatch : res) {
          return res;
        }
      }
      return returnState || returnStateOverride ? { isMatch: false, output: '' } : false;
    };
    arrayMatcher.isMatch = (input, returnStateOverride) => arrayMatcher(input, returnStateOverride);
    return arrayMatcher;
  }

  if (typeof glob !== 'string') {
    throw new TypeError('Expected a string or array of strings for glob pattern');
  }

  const matcher = (input, returnStateOverride = false) => {
    if (typeof input !== 'string') return false;
    const matched = callGoSync('isMatch', glob, input, options);
    if (returnState || returnStateOverride) {
      return { isMatch: matched, output: matched ? input : '' };
    }
    return matched;
  };

  matcher.isMatch = (input, returnStateOverride = false) => matcher(input, returnStateOverride);
  return matcher;
}

picomatch.isMatch = function(input, glob, options = {}) {
  if (Array.isArray(glob)) {
    return glob.some(g => picomatch.isMatch(input, g, options));
  }
  if (typeof glob !== 'string') {
    throw new TypeError('Expected a string or array of strings for glob pattern');
  }
  return callGoSync('isMatch', glob, input, options);
};

picomatch.scan = function(input, options = {}) {
  if (typeof input !== 'string') {
    throw new TypeError('Expected a string for scan input');
  }
  const res = callGoSync('scan', input, '', options);
  return res || { input, prefix: '', base: input, glob: '', isGlob: false };
};

picomatch.parse = function(input, options = {}) {
  if (typeof input !== 'string') {
    throw new TypeError('Expected a string for parse input');
  }
  const res = callGoSync('parse', input, '', options);
  return res || { input, output: input, isGlob: false };
};

picomatch.makeRe = function(input, options = {}) {
  const parsed = picomatch.parse(input, options);
  const flags = options && options.nocase ? 'i' : '';
  try {
    return new RegExp(parsed.output || input, flags);
  } catch (e) {
    return new RegExp(String(input).replace(/[.*+?^${}()|[\]\\]/g, '\\$&'), flags);
  }
};

picomatch.compileRe = function(state, options = {}) {
  const flags = options && options.nocase ? 'i' : '';
  return new RegExp(state.output || state.input || '', flags);
};

module.exports = picomatch;
