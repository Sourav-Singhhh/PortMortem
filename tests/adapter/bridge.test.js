const assert = require('assert');
const { GoBridge, PROTOCOL_VERSION, MSG_TYPES } = require('./bridge');

async function runTests() {
    console.log('Starting Production Refactored Bridge Tests...');
    
    // Ensure you have compiled the Go binary first:
    // go build -o bridge.exe main.go (Windows) or go build -o bridge main.go (Linux/Mac)
    const bridge = new GoBridge();

    try {
        console.log('Test 1: Startup Handshake (Ping/Pong)');
        await bridge.init();
        assert.strictEqual(bridge.isReady, true, 'Bridge should be ready after init');
        console.log('✅ Test 1 Passed\n');

        console.log('Test 2: Stress Test - 1000 Sequential Requests');
        const seqStart = Date.now();
        for (let i = 1; i <= 1000; i++) {
            const res = await bridge.request({ step: i, pattern: `*.v${i}` });
            assert.strictEqual(res.step, i);
            assert.strictEqual(res.pattern, `*.v${i}`);
        }
        console.log(`✅ Test 2 Passed in ${Date.now() - seqStart}ms\n`);

        console.log('Test 3: Stress Test - 1000 Concurrent Requests');
        const concStart = Date.now();
        const tasks = [];
        for (let i = 0; i < 1000; i++) {
            tasks.push(bridge.request({ idx: i, data: `load-${i}` }));
        }
        const results = await Promise.all(tasks);
        assert.strictEqual(results.length, 1000);
        for (let i = 0; i < 1000; i++) {
            assert.strictEqual(results[i].idx, i);
            assert.strictEqual(results[i].data, `load-${i}`);
        }
        console.log(`✅ Test 3 Passed in ${Date.now() - concStart}ms\n`);

        console.log('Test 4: Deterministic Timeout Handling');
        try {
            // Tell Go to sleep for 500ms, but set timeout to 100ms
            await bridge.simulateDelay(500, 100);
            assert.fail('Should have timed out');
        } catch (err) {
            assert.ok(err.message.includes('timed out'), `Expected timeout, got: ${err.message}`);
            console.log('✅ Test 4 Passed\n');
        }

        console.log('Test 5: Malformed JSON Request');
        const malformedPromise = new Promise((resolve) => {
            bridge.once('global_error', (errorMsg) => {
                assert.strictEqual(errorMsg, 'invalid json format');
                resolve();
            });
        });
        bridge.sendMalformedRaw('{ this is not valid json');
        await malformedPromise;
        console.log('✅ Test 5 Passed\n');

        console.log('Test 6: Message Field Validation (Missing Version/Type)');
        const missingFieldsPromise = new Promise((resolve) => {
            bridge.once('global_error', (errorMsg) => {
                assert.ok(errorMsg.includes('missing required fields'));
                resolve();
            });
        });
        // Bypass _sendRaw to send incomplete JSON payload without standard wrapping
        bridge.sendMalformedRaw(JSON.stringify({ id: "invalid-1", payload: {} }));
        await missingFieldsPromise;
        console.log('✅ Test 6 Passed\n');

        console.log('Test 7: Maximum Message Size Enforcement (>1MB)');
        try {
            // Create a payload larger than 1MB
            const hugeString = 'a'.repeat(1024 * 1024 + 10);
            await bridge.request({ data: hugeString });
            assert.fail('Should have rejected due to message size limit');
        } catch (err) {
            assert.ok(err.message.includes('exceeds maximum allowed size'));
            console.log('✅ Test 7 Passed\n');
        }

        console.log('Test 8: Invalid Responses from Go (Simulated)');
        const invalidRespPromise = new Promise((resolve) => {
            bridge.once('invalid_response', (line, err) => {
                assert.strictEqual(line, 'Not even json');
                assert.ok(err instanceof SyntaxError || err.message.includes('Missing required fields'));
                resolve();
            });
        });
        // Simulate receiving invalid JSON from stdout
        bridge._handleResponse('Not even json');
        await invalidRespPromise;
        console.log('✅ Test 8 Passed\n');

        console.log('Test 9: Graceful Shutdown with Awaiting Process Exit');
        await bridge.close();
        assert.strictEqual(bridge.isReady, false, 'Bridge should be marked as not ready');
        assert.ok(bridge.process.exitCode !== null || bridge.process.killed, 'Process should have exited cleanly');
        console.log('✅ Test 9 Passed\n');

        console.log('Test 10: Abrupt Process Termination');
        const tempBridge = new GoBridge();
        await tempBridge.init();
        const closePromise = new Promise((resolve) => {
            tempBridge.once('close', (code) => {
                assert.strictEqual(tempBridge.isReady, false);
                resolve();
            });
        });
        // Abruptly kill the Go subprocess
        tempBridge.process.kill('SIGTERM');
        await closePromise;
        console.log('✅ Test 10 Passed\n');

        console.log('ALL TESTS PASSED SUCCESSFULLY! 🎉');

    } catch (err) {
        console.error('❌ Test Failed:', err);
        console.error('Note: Ensure you built the Go binary before running this test:');
        console.error('  go build -o bridge.exe main.go');
        process.exitCode = 1;
    }
}

runTests();
