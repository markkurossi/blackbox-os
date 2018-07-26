//
// init.js
//
// Copyright (c) 2018 Markku Rossi
//
// All rights reserved.
//

function initJavaScript() {
    console.log("Booting...");

    if (!WebAssembly.instantiateStreaming) { // polyfill
        WebAssembly.instantiateStreaming = async (resp, importObject) => {
	    const source = await (await resp).arrayBuffer();
	    return await WebAssembly.instantiate(source, importObject);
        };
    }

    //document.addEventListener('keydown', function(ev) {
    //    console.log("keydown:", ev);
    //    if (keyboardHandler) {
    //        keyboardHandler(ev);
    //    }
    //})

    const go = new Go();
    let mod, inst;
    console.time("WebAssembly")
    WebAssembly.instantiateStreaming(fetch("kernel.wasm"), go.importObject)
        .then((result) => {
            mod = result.module;
            inst = result.instance;

            console.timeEnd("WebAssembly");
            async function run() {
                await go.run(inst);
                // uninit();
                // reset instance
                inst = await WebAssembly.instantiate(mod, go.importObject);
                console.log("Halted");
            }
            console.log("Running")
            run();
        });
}

function init(keyboard, mouse, input) {
    keyboardHandler = keyboard;
    mouseHandler = mouse;
    inputHandler = input;
}
function uninit() {
    keyboardHandler = undefined;
    mouseHandler = undefined;
}
