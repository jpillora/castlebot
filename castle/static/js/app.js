var module = angular.module("castle", []);

module.controller("AppController", function($scope) {
  var app = window.app = $scope.app = this;
  app.connected = false;
  app.data = $scope.data = {};
  var conn = velox("/sync", app.data);
  conn.onupdate = function() {
    $scope.$apply();
  };
  conn.onchange = function(connected) {
    app.connected = connected;
    $scope.$apply();
  };

  $scope.defined = function(obj) {
    return obj && typeof obj === "object" && Object.keys(obj).length > 0;
  };
});

module.directive("blobSrc", function() {
  return {
    restrict: "A",
    link: function(scope, elem, attrs) {
      window.blobSrc = scope;
      var e = elem[0];
      scope.$watch(attrs.blobSrc, function(blob) {
        if (blob) e.src = URL.createObjectURL(blob);
      });
    }
  };
});

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

module.directive("since", function(sinceMillis) {
  return {
    restrict: "A",
    link: function(s, e, attrs) {
      var d, t;
      var check = function() {
        clearTimeout(t);
        if (d && !isNaN(d) && d instanceof Date) {
          var millis = +new Date() - d;
          e.text(sinceMillis(millis) + ("ago" in attrs ? " ago" : ""));
          if (millis < 60 * 1000) {
            t = setTimeout(check, 1000);
          } else if (millis < 60 * 60 * 1000) {
            t = setTimeout(check, 60 * 1000);
          }
        }
      };
      s.$watch(attrs.since, function(s) {
        d = new Date(s);
        e.attr("title", d.toString());
        check();
      });
    }
  };
});
