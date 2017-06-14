module.controller("ScannerController", function($scope, $http, $timeout) {
  var scanner = ($scope.scanner = window.scanner = this);
  scanner.data = {};
  scanner.hosts = [];
  scanner.hostMap = {};
  $scope.$watch(
    "app.data.modules.scanner",
    function(data) {
      scanner.data = data || {};
      var status = scanner.data.status || {};
      if (status.hosts) {
        extractHosts(status.hosts);
      }
    },
    true
  );

  $scope.nano = function(nano) {
    var ms = nano / 1e6;
    return Math.round(ms) + "ms";
  };

  $scope.hasKeys = function(obj) {
    return obj && typeof obj === "object" && Object.keys(obj).length > 0;
  };

  var ipToInt = function(ip) {
    var int = 0;
    var octs = ip.split(".");
    for (var i = 0; i < octs.length; i++) {
      var o = parseInt(octs[i]);
      int = (int << 8) + o;
    }
    return int;
  };

  var extractHosts = function(hosts) {
    var now = +new Date();
    //loop
    for (var key in hosts) {
      var updates = hosts[key];
      var h = scanner.hostMap[key];
      if (!h) {
        h = {};
        scanner.hosts.push(h);
        scanner.hostMap[key] = h;
      }
      angular.merge(h, updates);
      //computed properties
      var diff = now - new Date(h.seenAt);
      var mins = diff / 1000 / 60;
      if (mins <= 10) {
        h.$class = "positive";
        h.$classn = 1;
      } else if (mins <= 30) {
        h.$class = "warning";
        h.$classn = 2;
      } else {
        h.$class = "negative";
        h.$classn = 3;
      }
      h.$seenMins = mins;
    }
    //sort
    scanner.hosts.sort(function(a, b) {
      if (a.$classn !== b.$classn) {
        return a.$classn < b.$classn ? -1 : 1;
      }
      return ipToInt(a.ip) < ipToInt(b.ip) ? -1 : 1;
    });
    //angular digest
  };
});
