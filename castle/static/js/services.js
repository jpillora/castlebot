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
