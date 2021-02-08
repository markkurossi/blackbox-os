//
// init.js
//
// Copyright (c) 2018-2021 Markku Rossi
//
// All rights reserved.
//

var keyboardHandler;
var display;
var loader;

function initJavaScript(displayId) {
    display = new Display(document.getElementById(displayId));
    loader = document.getElementById('loader');

    console.log("Booting...");
    loader.style.display = 'block';

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
    if (false) {
        document.addEventListener('keyup', function(ev) {
            if (ev.metaKey) {
                return;
            }
            if (keyboardHandler) {
                keyboardHandler(ev);
            }
        })
    }

    const go = new Go();
    let mod, inst;
    console.time("WebAssembly")
    WebAssembly.instantiateStreaming(fetch("kernel.wasm"), go.importObject)
        .then((result) => {
            mod = result.module;
            inst = result.instance;

            console.timeEnd("WebAssembly");
            loader.style.display = 'none';
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

function initKeyboard(keyboard) {
    keyboardHandler = keyboard;
}

function init(keyboard, mouse, input) {
    keyboardHandler = keyboard;
}

function uninit() {
    keyboardHandler = undefined;
}

/*********************************** XXX ************************************/

function spawn() {
    const worker = new Worker("process.js?_ts=" + new Date().getTime());

    fetch("bin/echo.wasm").then(response =>
        response.arrayBuffer()
    ).then(bytes =>
        worker.postMessage({
            command: "init",
            data: bytes
        })
    )
}
