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
        cmd: "write",
        fd: fd,
        data: buf,
        offset: offset,
        length: length,
    }, {
        cb: callback
    });
}

function syscall_read(fd, buf, offset, length, callback) {
    syscall({
        cmd: "read",
        fd: fd,
        length: length
    }, {
        cb: callback,
        buf: buf,
        offset: offset
    })
}

let syscall_id = 1;
let syscall_pending = new Map();

function syscall(params, context) {
    params.id = syscall_id++;
    syscall_pending.set(params.id, context);
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
    console.log("process:", e.data);
    switch (e.data.cmd) {
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
        let ctx = syscall_pending.get(e.data.id);
        if (!ctx) {
            console.error("unknown syscall result: id=%d", e.data.id);
        } else {
            syscall_pending.delete(e.data.id);

            let err = null;
            if (e.data.error) {
                err = new Error(e.data.error);
                err.code = e.data.error;
            }
            if (e.data.buf && ctx.buf) {
                ctx.buf.set(e.data.buf, ctx.offset || 0);
            }

            ctx.cb(err, e.data.code, e.data.buf);
        }
        break;

    default:
        console.error("unknown command:", e);
    }
}
