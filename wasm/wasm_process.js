//
// Copyright (c) 2021 Markku Rossi
//
// All rights reserved.
//

console.log("wasm_process.js: global:", global);

function timeout(ms) {
    return new Promise(resolve => setTimeout(resolve, ms));
}

global.process = {
    getuid() { return -1; },
    getgid() { return -1; },
    geteuid() { return -1; },
    getegid() { return -1; },
    getgroups() { throw enosys(); },
    pid: -1,
    ppid: -1,
    umask() { throw enosys(); },
    async cwd() {
        console.log("global.process.cwd...")
        await timeout(1000);
        console.log("global.process.cwd: done")
        throw enosys();
    },
    chdir() { throw enosys(); },
}
