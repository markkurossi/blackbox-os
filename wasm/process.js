//
// process.js
//
// Copyright (c) 2021 Markku Rossi
//
// All rights reserved.
//

importScripts('wasm_exec.js');
importScripts('wasm_fs.js');

const utf8Encode = new TextEncoder();

function syscall_write(fd, buf, offset, length, callback) {
    syscall({
        type: "write",
        fd: fd,
        data: buf,
        offset: offset,
        length: length,
    }, callback);
}

let syscall_id = 1;
let syscall_pending = new Map();

function syscall(params, callback) {
    params.id = syscall_id++;
    syscall_pending.set(params.id, callback);
    postMessage(params);
}

onmessage = function(e) {
    try {
        processEvent(e);
    } catch (error) {
        console.error(error);
    }
}

function processEvent(e) {
    console.log("processEvent:", e.data);
    switch (e.data.command) {
    case "init":
        let go = new Go();

        go.argv = e.data.argv || ["wasm"];

        let mod, inst;
        console.time("WebAssembly")
        WebAssembly.instantiate(e.data.code, go.importObject)
            .then((result) => {
                mod = result.module;
                inst = result.instance;

                console.timeEnd("WebAssembly");
                async function run() {
                    await go.run(inst);
                    // reset instance
                    inst = await WebAssembly.instantiate(mod, go.importObject);
                    console.log("halted");
                }
                console.log("running")
                run();
            });
        break;

    case "result":
        let cb = syscall_pending.get(e.data.id);
        if (!cb) {
            console.error("unknown syscall result:", e.data.id);
        } else {
            syscall_pending.delete(e.data.id);
            cb(e.data.error, e.data.code);
        }
        break;

    default:
        console.error("unknown command:", e);
    }
}
