module.service("since", function(sinceMillis) {
  return function(date) {
    if (!date || !(date instanceof Date)) return "-";
    return sinceMillis(+new Date() - date);
  };
});

module.service("sinceMillis", function() {
  var scale = [
    ["ms", 1000],
    ["sec", 60],
    ["minute", 60],
    ["hour", 24],
    ["day", 31],
    ["month", 12],
    ["year", 10],
    ["decade", 10],
    ["century", 10]
  ];
  return function(millis) {
    var v = millis;
    if (v < 0) return "<future>";
    var s;
    for (var i = 0; i < scale.length; i++) {
      s = scale[i];
      if (v < s[1]) break;
      v = Math.round(v / s[1]);
    }
    return v + " " + s[0] + (v === 1 ? "" : "s");
  };
});

module.service("scale", function() {
  return function(n, p) {
    // set defaults
    if (typeof n !== "number" || isNaN(n)) n = 0;
    if (!p || typeof p !== "number") p = 2;
    // set scale index 1000,100000,... becomes 1,2,...
    var i = Math.floor(Math.floor(Math.log10(n)) / 3);
    // reduce by scale
    var s = (n / Math.pow(10, i * 3)).toPrecision(p);
    // concat (no trailing 0s) and choose scale letter
    return s.toString().replace(/\.?0+$/, "") +
      " " +
      ["", "K", "M", "G", "T", "P", "Z"][i];
  };
});

module.filter("scale", function(scale) {
  return scale;
});
