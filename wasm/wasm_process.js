//
// Copyright (c) 2021 Markku Rossi
//
// All rights reserved.
//

console.log("wasm_process.js: global:", global);

global.process = {
    __cwd: "/",

    getuid() { return -1; },
    getgid() { return -1; },
    geteuid() { return -1; },
    getegid() { return -1; },
    getgroups() { throw enosys(); },
    pid: -1,
    ppid: -1,
    umask() { throw enosys(); },
    cwd() {
        return global.process.__cwd;
    },
    chdir() { throw enosys(); },
}

function syscallSetWD(cwd) {
    global.process.__cwd = cwd;
}
