//
// init.js
//
// Copyright (c) 2018 Markku Rossi
//
// All rights reserved.
//

var keyboardHandler;
var display;

function initJavaScript(displayId) {
    display = new Display(document.getElementById(displayId));

    console.log("Booting...");

    if (!WebAssembly.instantiateStreaming) { // polyfill
        WebAssembly.instantiateStreaming = async (resp, importObject) => {
	    const source = await (await resp).arrayBuffer();
	    return await WebAssembly.instantiate(source, importObject);
        };
    }

    document.addEventListener('keydown', function(ev) {
        if (ev.metaKey) {
            return;
        }
        if (keyboardHandler) {
            keyboardHandler(ev);
        }
    })
    document.addEventListener('keyup', function(ev) {
        if (ev.metaKey) {
            return;
        }
        if (keyboardHandler) {
            keyboardHandler(ev);
        }
    })

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
                uninit();
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
}

function uninit() {
    keyboardHandler = undefined;
}

function displayWidth() {
    return display.width;
}

function displayHeight() {
    return display.height;
}

function displayClear() {
    display.clear();
}

function displayAddLine(line) {
    display.addLine(line);
}
