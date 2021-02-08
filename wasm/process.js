//
// process.js
//
// Copyright (c) 2021 Markku Rossi
//
// All rights reserved.
//

importScripts('wasm_exec.js');

onmessage = function(e) {
    try {
        processEvent(e);
    } catch (error) {
        console.error(error);
    }
}

function processEvent(e) {
    console.log("Process:", e);
    switch (e.data.command) {
    case "init":
        const go = new Go();
        let mod, inst;
        console.time("WebAssembly")
        WebAssembly.instantiate(e.data.data, go.importObject)
            .then((result) => {
                mod = result.module;
                inst = result.instance;

                console.timeEnd("WebAssembly");
                async function run() {
                    await go.run(inst);
                    // reset instance
                    inst = await WebAssembly.instantiate(mod, go.importObject);
                    console.log("Halted");
                }
                console.log("Running")
                run();
            });
        break;

    default:
        console.error("Unknown command:", e.command);
    }
}
