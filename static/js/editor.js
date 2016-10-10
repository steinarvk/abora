var canvas = new fabric.Canvas("canvas");
canvas.selection = false;

var selectedItem = null;
var drawingLineseg = null;
var extendingLineseg = null;

function createLineseg(x, y) {
  var rv = {
    currentColour: "red",
  };

  var alwaysForward = true;

  var dots = [];
  var lines = [];
  dots.push(new fabric.Circle({
    radius: 10,
    fill: rv.currentColour,
    originX: "center",
    originY: "center",
    selectable: false,
    left: x,
    top: y,
  }));

  rv.addPoint = function(nx, ny) {
    var x = dots[dots.length-1].left;
    var y = dots[dots.length-1].top;
    if (alwaysForward && nx < x) {
      nx = x;
    }
    dots.push(new fabric.Circle({
      radius: 10,
      fill: rv.currentColour,
      originX: "center",
      originY: "center",
      selectable: false,
      left: nx,
      top: ny,
    }));
    lines.push(new fabric.Line([x, y, nx, ny], {
      strokeWidth: 10,
      stroke: rv.currentColour,
      selectable: false,
      originX: "center",
      originY: "center",
    }));
    dots[dots.length-1].aboraObject = rv;
    lines[lines.length-1].aboraObject = rv;
    canvas.add(dots[dots.length-1]);
    canvas.add(lines[lines.length-1]);
  }

  rv.setEndpoint = function(nx, ny) {
    var oldX = dots[dots.length-2].left;
    if (alwaysForward && nx < oldX) {
      nx = oldX;
    }
    dots[dots.length-1].left = nx;
    dots[dots.length-1].top = ny;
    lines[lines.length-1].set("x2", nx);
    lines[lines.length-1].set("y2", ny);
  }

  rv.setColour = function(col) {
    rv.currentColour = col;
    dots.forEach(function(dot) {
      dot.set("fill", col);
    });
    lines.forEach(function(line) {
      line.set("stroke", col);
    });
  }

  rv.remove = function() {
    dots.forEach(function(dot) {
      dot.remove();
    });
    lines.forEach(function(line) {
      line.remove();
    });
  }

  rv.setUnselected = function() {
    rv.setColour("blue");
    selectedItem = null;
  }

  rv.removeLastPoint = function() {
    if (dots.length >= 3) {
      canvas.remove(dots.pop());
      canvas.remove(lines.pop());
    }
  }

  rv.setSelected = function() {
    if (selectedItem) {
      selectedItem.setUnselected();
    }
    selectedItem = rv;
    rv.setColour("purple");
  };

  dots.forEach(function(x) {
    canvas.add(x);
    x.aboraObject = rv;
  });
  lines.forEach(function(x) {
    canvas.add(x);
    x.aboraObject = rv;
  });

  rv.addPoint(x, y);

  return rv;
}

canvas.on("mouse:down", function(option) {
  console.log(option);
  var x = option.e.offsetX, y = option.e.offsetY;
  if (extendingLineseg && option.e.ctrlKey) {
    console.log("handling extendingLineSeg");
    extendingLineseg.addPoint(x, y);
    return;
  }
  if (extendingLineseg) {
    console.log("handling extendedLineSeg");
    extendingLineseg.setUnselected();
    extendingLineseg = null;
    return;
  }

  if (!option.target) {
    console.log("handling clickOnEmptySpace");
    extendingLineseg = createLineseg(x, y);
  } else if (option.target.aboraObject) {
    console.log("handling selection");
    option.target.aboraObject.setSelected();
    option.target.aboraObject.addPoint(x, y);
    extendingLineseg = option.target.aboraObject;
  }

  canvas.renderAll();
});

canvas.on("mouse:move", function(option) {
  if (drawingLineseg) {
    drawingLineseg.setEndpoint(option.e.offsetX, option.e.offsetY);
  } else if(extendingLineseg) {
    extendingLineseg.setEndpoint(option.e.offsetX, option.e.offsetY);
  }

  canvas.renderAll();
});

canvas.on("mouse:up", function(option) {
  if (drawingLineseg) {
    drawingLineseg.setColour("blue");
    drawingLineseg = null;
  }

  canvas.renderAll();
});

document.addEventListener("keydown", function(evt) {
  var x = evt.offsetX, y = evt.offsetY;

  if (evt.key === "Delete") {
    if (selectedItem) {
      selectedItem.remove();
      selectedItem = null;
      extendingLineseg = null;
    }
  }
  if (evt.key === "Backspace") {
    if (selectedItem) {
      selectedItem.removeLastPoint();
      extendingLineseg = null;
      return;
    }
    if (extendingLineseg) {
      extendingLineseg.removeLastPoint();
      extendingLineseg.setEndpoint(x, y);
      return;
    }
  }

  console.log(evt);
  canvas.renderAll();
}, false);

canvas.setBackgroundImage('/tmp/spec2.png', canvas.renderAll.bind(canvas));


