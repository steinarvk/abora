$(function() {

var canvas = new fabric.Canvas("canvas");
canvas.selection = false;

var selectedItem = null;
var drawingLineseg = null;
var extendingLineseg = null;

var fullModel = [];

function createLineseg(x, y) {
  var xTrans = 0;

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
    console.log("removing element. There are " + fullModel.length + " elements");
    dots.forEach(function(dot) {
      dot.remove();
    });
    lines.forEach(function(line) {
      line.remove();
    });
    fullModel = _.without(fullModel, rv);
    console.log("removed element. There are " + fullModel.length + " elements");
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

  function fromPixelspaceX(trans, x) {
    return x * trans.xMul + trans.xAdd;
  }

  function fromPixelspaceY(trans, y) {
    return y * trans.yMul + trans.yAdd;
  }

  rv.setTransformation = function(oldTrans, newTrans) {
    var nxAdd = newTrans.xAdd, nxMul = newTrans.xMul;
    var nyAdd = newTrans.yAdd, nyMul = newTrans.yMul;
    console.log("setting trans " + nxAdd);
    console.log("setting trans " + nxMul);
    console.log("setting trans " + nyAdd);
    console.log("setting trans " + nyMul);
    function transformX(x) {
      return (fromPixelspaceX(oldTrans, x) - nxAdd) / nxMul;
    }
    function transformY(y) {
      return (fromPixelspaceY(oldTrans, y) - nyAdd) / nyMul;
    }
    dots.forEach(function(dot) {
      console.log("dot was at x " + dot.get("left"));
      console.log("dot was at y " + dot.get("top"));
      console.log("dot was at time " + fromPixelspaceX(oldTrans, dot.get("left")));
      console.log("dot was at freq " + fromPixelspaceX(oldTrans, dot.get("top")));
      dot.set("left", transformX(dot.get("left")));;
      dot.set("top", transformY(dot.get("top")));;
    });
    lines.forEach(function(line) {
      line.set("x1", transformX(line.get("x1")));
      line.set("x2", transformX(line.get("x2")));
      line.set("y1", transformY(line.get("y1")));
      line.set("y2", transformY(line.get("y2")));
    });
    dots.forEach(function(dot) {
      console.log("dot is at x " + dot.get("left"));
      console.log("dot is at y " + dot.get("top"));
      console.log("dot is at time " + fromPixelspaceX(newTrans, dot.get("left")));
      console.log("dot is at freq " + fromPixelspaceX(newTrans, dot.get("top")));
    });
  }

  rv.setSelected = function() {
    if (selectedItem) {
      selectedItem.setUnselected();
    }
    selectedItem = rv;
    rv.setColour("purple");
  };

  rv.toStringForm = function(trans) {
    var rv = [];
    dots.forEach(function(dot) {
      var t = fromPixelspaceX(trans, dot.get("left"));
      var freq = fromPixelspaceY(trans, dot.get("top"));
      rv.push("" + t + ":" + freq);
    });
    return rv.join("-");
  }

  dots.forEach(function(x) {
    canvas.add(x);
    x.aboraObject = rv;
  });
  lines.forEach(function(x) {
    canvas.add(x);
    x.aboraObject = rv;
  });

  rv.addPoint(x, y);

  fullModel.push(rv);

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

function makeParams(offset, duration) {
  var w = canvas.width, h = canvas.height;
  var params = {
    pxWidth: w,
    pxHeight: h,
    duration: duration,
    t: offset,
  };
  return params;
}

function withMetadata(params, f) {
  $.get("/spectrogram/metadata", params).done(function(data) {
    f(data);
  });
}

function displayBackgroundSpectrogram(params) {
  var imageUrl = "/spectrogram/png?" + $.param(params);
  canvas.setBackgroundImage(imageUrl, canvas.renderAll.bind(canvas));
}

var currentOffset = 0;
var currentDuration = 1000;

var transformation = null;

function refreshView() {
  var params = makeParams(currentOffset, currentDuration);
  withMetadata(params, function(metadata) {
    var newTrans = {
      xAdd: currentOffset,
      xMul: 1.0 / metadata.TimeResolution,
      yAdd: metadata.HighFrequency,
      yMul: (metadata.LowFrequency - metadata.HighFrequency) / metadata.FrequencyBuckets
    };
    fullModel.forEach(function(x) {
      x.setTransformation(transformation, newTrans);
    });
    transformation = newTrans;
    canvas.renderAll();
  });
  displayBackgroundSpectrogram(params);
  canvas.renderAll();
}

$("body").append($("<button/>").text("Forward").click(function() {
  currentOffset += 2;
  refreshView();
}));

$("body").append($("<button/>").text("Back").click(function() {
  currentOffset -= 2;
  if (currentOffset < 0) {
    currentOffset = 0;
  }
  refreshView();
}));

$("body").append($("<textarea id='dump'/>"));

$("body").append($("<button/>").text("Dump").click(function() {
  console.log("dump button was clicked");
  var rv = [];
  fullModel.forEach(function(x) {
    rv.push(x.toStringForm(transformation));
  });
  $("#dump").text(rv.join("\n"));
}));

refreshView();

});
