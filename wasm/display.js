//
// display.js
//
// Copyright (c) 2018 Markku Rossi
//
// All rights reserved.
//

function Display(element) {
    var self = this;

    self.element = element;
    self.measure();
}

Display.prototype.measure = function() {
    var txt = "Black Box OS 2018!";

    var line = document.createElement('div');
    var span = document.createElement('span');
    span.appendChild(document.createTextNode(txt));
    line.appendChild(span);

    this.element.appendChild(line);

    this.charWidth = span.offsetWidth / txt.length;
    this.charHeight = span.offsetHeight;

    this.computeSize();
    this.clear();
}

Display.prototype.computeSize = function() {
    var padding = 2 * 10;
    this.widthPx = this.element.offsetWidth - padding;
    this.heightPx = this.element.offsetHeight;

    this.width = parseInt(this.widthPx / this.charWidth);
    this.height = parseInt(this.heightPx / this.charHeight);

    console.log("Display: " + this.width + "x" + this.height);
}

Display.prototype.clear = function() {
    while (this.element.firstChild)
        this.element.removeChild(this.element.firstChild);
}

Display.prototype.addLine = function(data) {
    var i;
    var line = new Line();

    for (i = 0; i < data.length; i += 4) {
        line.add(data[i], data[i + 1], data[i + 2], data[i + 3]);
    }
    line.flush();

    this.element.appendChild(line.el);
}

function Line() {
    this.el = document.createElement('div');
    this.txt = '';
    this.fg = 0;
    this.bg = 0;
    this.flags = 0;
}

Line.prototype.add = function(code, fg, bg, flags) {
    if (this.fg != fg || this.bg != bg || this.flags != flags) {
        this.flush();
        this.fg = fg;
        this.bg = bg;
        this.flags = flags;
    }
    this.txt += String.fromCharCode(code);
}

Line.prototype.flush = function() {
    if (this.txt.length == 0) {
        return;
    }
    var span = document.createElement('span');
    if (this.flags != 0) {
        span.style.backgroundColor = '#aaa';
    }
    span.appendChild(document.createTextNode(this.txt));

    this.el.appendChild(span)

    this.txt = '';
}
