const picomatch = require('../../original-picomatch/picomatch-master');
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

    const { id, pattern, input, options } = payload;
    try {
        const isMatch = picomatch(pattern, options || {})(input);
        console.log(JSON.stringify({ id, success: true, result: isMatch }));
    } catch (err) {
        console.log(JSON.stringify({ id, success: false, error: err.message }));
    }
});
