const { spawn } = require('child_process');
const readline = require('readline');
const path = require('path');
const EventEmitter = require('events');

const PROTOCOL_VERSION = '1.0';
const MAX_MESSAGE_SIZE_BYTES = 1024 * 1024; // 1MB maximum supported message size

// Shared constants matching Go message types
const MSG_TYPES = Object.freeze({
    HANDSHAKE: 'handshake',
    REQUEST: 'request',
    RESPONSE: 'response',
    SHUTDOWN: 'shutdown',
    SIMULATE_DELAY: 'simulate_delay',
    ERROR: 'error'
});

class GoBridge extends EventEmitter {
    constructor(goBinaryPath) {
        super();
        this.goBinaryPath = goBinaryPath || this._defaultBinaryPath();
        this.process = null;
        this.rl = null;
        this.pendingRequests = new Map();
        this.messageIdCounter = 1;
        this.isReady = false;
        this.isClosing = false;
    }

    _defaultBinaryPath() {
        const ext = process.platform === 'win32' ? '.exe' : '';
        return path.join(__dirname, `bridge${ext}`);
    }

    /**
     * Start the process and perform a startup handshake
     * @returns {Promise<void>}
     */
    async init() {
        if (this.process && !this.process.killed && this.process.exitCode === null) {
            return;
        }

        this.isClosing = false;
        this.process = spawn(this.goBinaryPath, [], { stdio: ['pipe', 'pipe', 'pipe'] });

        this.rl = readline.createInterface({
            input: this.process.stdout,
            terminal: false
        });

        this.rl.on('line', (line) => {
            this._handleResponse(line);
        });

        this.process.stderr.on('data', (data) => {
            this.emit('error', new Error(`Go Stderr: ${data.toString()}`));
        });

        this.process.on('close', (code) => {
            this.isReady = false;
            this.isClosing = false;
            this._rejectAll(new Error(`Go process exited with code ${code !== null ? code : 'unknown'}`));
            this.emit('close', code);
        });

        this.process.on('error', (err) => {
            this.isReady = false;
            this.isClosing = false;
            this._rejectAll(new Error(`Failed to start Go process: ${err.message}`));
            this.emit('error', err);
        });

        // Perform startup handshake (ping/pong)
        const handshakeRes = await this._sendRaw({
            type: MSG_TYPES.HANDSHAKE
        }, 5000);

        if (handshakeRes.type !== MSG_TYPES.HANDSHAKE || handshakeRes.payload?.status !== 'ready') {
            throw new Error('Handshake failed: unexpected response from Go bridge');
        }

        this.isReady = true;
    }

    _handleResponse(line) {
        try {
            const response = JSON.parse(line);
            
            // Validate required fields on incoming message
            if (!response || typeof response !== 'object' || !response.version || !response.type) {
                this.emit('invalid_response', line, new Error('Missing required fields: version and type'));
                return;
            }

            if (response.version !== PROTOCOL_VERSION) {
                this.emit('invalid_response', line, new Error(`Protocol version mismatch: got ${response.version}`));
                return;
            }

            if (response.id && this.pendingRequests.has(response.id)) {
                const { resolve, reject, timer } = this.pendingRequests.get(response.id);
                clearTimeout(timer);
                this.pendingRequests.delete(response.id);

                if (response.error || response.type === MSG_TYPES.ERROR) {
                    reject(new Error(response.error || 'Unknown error received from Go'));
                } else {
                    resolve(response);
                }
            } else if (response.error || response.type === MSG_TYPES.ERROR) {
                // Handle global syntax or protocol validation errors from Go
                this.emit('global_error', response.error, response);
            } else {
                this.emit('unsolicited_response', response);
            }
        } catch (err) {
            this.emit('invalid_response', line, err);
        }
    }

    _rejectAll(error) {
        for (const [id, { reject, timer }] of this.pendingRequests.entries()) {
            clearTimeout(timer);
            reject(error);
        }
        this.pendingRequests.clear();
    }

    _sendRaw(msg, timeoutMs = 5000) {
        return new Promise((resolve, reject) => {
            if (this.isClosing && msg.type !== MSG_TYPES.SHUTDOWN) {
                return reject(new Error('Bridge is currently closing; no new requests accepted'));
            }
            if (!this.process || this.process.killed || this.process.exitCode !== null) {
                return reject(new Error('Go process is not running'));
            }

            const id = msg.id || `msg-${this.messageIdCounter++}`;
            const payload = { id, version: PROTOCOL_VERSION, ...msg };

            const serialized = JSON.stringify(payload) + '\n';
            if (Buffer.byteLength(serialized, 'utf8') > MAX_MESSAGE_SIZE_BYTES) {
                return reject(new Error(`Message exceeds maximum allowed size of ${MAX_MESSAGE_SIZE_BYTES} bytes`));
            }

            const timer = setTimeout(() => {
                if (this.pendingRequests.has(id)) {
                    this.pendingRequests.delete(id);
                    reject(new Error(`Request ${id} timed out after ${timeoutMs}ms`));
                }
            }, timeoutMs);

            this.pendingRequests.set(id, { resolve, reject, timer });

            this.process.stdin.write(serialized, (err) => {
                if (err) {
                    clearTimeout(timer);
                    this.pendingRequests.delete(id);
                    reject(err);
                }
            });
        });
    }

    /**
     * Send a standard request payload to the Go process
     * @param {Object} payload The data to send
     * @param {number} timeoutMs Timeout in milliseconds
     * @returns {Promise<Object>} The response payload from Go
     */
    async request(payload, timeoutMs = 5000) {
        if (!this.isReady) {
            throw new Error('Bridge not initialized. Call init() first.');
        }
        const response = await this._sendRaw({
            type: MSG_TYPES.REQUEST,
            payload
        }, timeoutMs);
        return response.payload;
    }

    /**
     * Send a simulate_delay message for deterministic testing
     */
    async simulateDelay(ms, timeoutMs = 5000) {
        if (!this.isReady) {
            throw new Error('Bridge not initialized.');
        }
        return this._sendRaw({
            type: MSG_TYPES.SIMULATE_DELAY,
            payload: { ms }
        }, timeoutMs);
    }

    /**
     * Send raw string to Go process stdin (useful for testing malformed JSON)
     */
    sendMalformedRaw(rawString) {
        if (this.process?.stdin && !this.isClosing) {
            this.process.stdin.write(rawString + '\n');
        }
    }

    /**
     * Gracefully shutdown the bridge, waiting for the Go process to fully exit
     */
    async close() {
        if (!this.process || this.process.killed || this.process.exitCode !== null) {
            this.isReady = false;
            this.isClosing = false;
            return;
        }

        this.isClosing = true;

        const exitPromise = new Promise((resolve) => {
            const timeout = setTimeout(() => {
                // If it doesn't close within 3 seconds, kill forcefully to avoid hangs
                if (this.process && this.process.exitCode === null) {
                    this.process.kill('SIGTERM');
                }
                resolve();
            }, 3000);

            this.once('close', () => {
                clearTimeout(timeout);
                resolve();
            });
        });

        try {
            await this._sendRaw({ type: MSG_TYPES.SHUTDOWN }, 2000);
        } catch (e) {
            // Ignore socket/pipe errors during shutdown attempt
        } finally {
            this.isReady = false;
            if (this.process.stdin && !this.process.stdin.destroyed) {
                this.process.stdin.end();
            }
        }

        // Explicitly await Go process exit
        await exitPromise;
        this._rejectAll(new Error('Bridge closed'));
    }
}

module.exports = { GoBridge, PROTOCOL_VERSION, MSG_TYPES, MAX_MESSAGE_SIZE_BYTES };
