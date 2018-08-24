//
// net.js
//
// Copyright (c) 2018 Markku Rossi
//
// All rights reserved.
//

var ST_WEBSOCKET	= 0;
var ST_CONNECTED	= 1;
var ST_CLOSED		= 2;

function WS(url, onOpen, onMessage, onError, onClose) {
    var self = this;

    self.url = url;
    self.goOnOpen = onOpen;
    self.goOnMessage = onMessage;
    self.goOnError = onError;
    self.goOnClose = onClose;

    self.state = ST_WEBSOCKET;

    self.ws = new WebSocket(url);
    self.ws.binaryType = 'arraybuffer';

    self.ws.onopen = function(evt) {
        self.onOpen(evt);
    }
    self.ws.onmessage = function(evt) {
        self.onMessage(evt);
    }
    self.ws.onerror = function(evt) {
        self.onError(evt);
    }
    self.ws.onclose = function(evt) {
        self.onClose(evt);
    }
}

WS.prototype.onOpen = function(evt) {
    this.state = ST_CONNECTED;
    this.goOnOpen(evt);
}

WS.prototype.onMessage = function(evt) {
    if (evt.data instanceof ArrayBuffer) {
        let dv = new DataView(evt.data);
        let result = [];

        for (var i = 0; i < dv.byteLength; i++) {
            result.push(dv.getUint8(i));
        }
        this.goOnMessage(result);
    } else {
        console.log("ws.onmessage:", evt);
    }
}

WS.prototype.onError = function(evt) {
    switch (this.state) {
    case ST_WEBSOCKET:
        this.goOnError("WebSocket connect failed");
        break;

    default:
        this.goOnError("Connection closed");
        break;
    }
}

WS.prototype.onClose = function(evt) {
    this.goOnClose(evt);
}

WS.prototype.send = function(data) {
    this.ws.send(data);
}

WS.prototype.close = function() {
    this.ws.onopen = undefined;
    this.ws.onmessage = undefined;
    this.ws.onerror = undefined;
    this.ws.onclose = undefined;
    this.ws.close();
}

function webSocketNew(url, onOpen, onMessage, onError, onClose) {
    return new WS(url, onOpen, onMessage, onError, onClose);
}

function webSocketSend(ws, data) {
    ws.send(data);
}

function webSocketClose(ws) {
    ws.close();
}

function reqListener () {
    console.log(this.responseText);
}

function httpGet(url) {
    var oReq = new XMLHttpRequest();
    oReq.addEventListener("load", reqListener);
    oReq.addEventListener("error", reqListener);
    oReq.addEventListener("close", reqListener);
    oReq.open("GET", url);
    oReq.send();
}
