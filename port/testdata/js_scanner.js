const scan = require('../../original-picomatch/picomatch-master/lib/scan');
const readline = require('readline');

const rl = readline.createInterface({
    input: process.stdin,
    output: process.stdout,
    terminal: false
});

rl.on('line', (line) => {
    if (!line.trim()) return;
    let payload;
    try {
        payload = JSON.parse(line);
    } catch (e) {
        console.log(JSON.stringify({ success: false, error: 'Malformed JSON input' }));
        return;
    }

    const { id, pattern, options } = payload;
    try {
        const res = scan(pattern, options || {});

        // Sanitize IEEE-754 Infinity into -1 for standard JSON serialization
        if (res.maxDepth === Infinity) {
            res.maxDepth = -1;
        }
        if (Array.isArray(res.tokens)) {
            res.tokens.forEach(t => {
                if (t.depth === Infinity) {
                    t.depth = -1;
                }
            });
        }

        // Normalize undefined slices to empty arrays for direct comparison with Go slices
        if (!res.slashes) res.slashes = [];
        if (!res.parts) res.parts = [];
        if (!res.tokens) res.tokens = [];

        console.log(JSON.stringify({ id, success: true, result: res }));
    } catch (err) {
        console.log(JSON.stringify({ id, success: false, error: err.message }));
    }
});
