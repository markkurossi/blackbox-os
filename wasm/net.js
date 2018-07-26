//
// net.js
//
// Copyright (c) 2018 Markku Rossi
//
// All rights reserved.
//

function webSocketNew(url, onMessage) {
    let ws = new WebSocket(url);

    ws.binaryType = 'arraybuffer';

    ws.onmessage = function(evt) {
        if (evt.data instanceof ArrayBuffer) {
            let dv = new DataView(evt.data);
            let result = [];

            for (var i = 0; i < dv.byteLength; i++) {
                result.push(dv.getUint8(i));
            }
            onMessage(result)
        } else {
            console.log("ws.onmessage:", evt, evt.data instanceof ArrayBuffer);
        }
    }

    return ws;
}

function webSocketSend(ws, data) {
    ws.send(data);
}

function webSocketClose(ws) {
    ws.onmessage = undefined;
    ws.close();
}
