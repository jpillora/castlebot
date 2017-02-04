module.controller("ScannerController", function($scope, $http, $timeout) {
  var scanner = $scope.scanner = window.scanner = this;
  scanner.data = {};
  $scope.$watch("app.data.modules.scanner", function(data) {
    scanner.data = data || {};
  });

  $scope.nano = function(nano) {
    var ms = nano / 1e6;
    return Math.round(ms) + "ms";
  };

  $scope.hasKeys = function(obj) {
    return obj && typeof obj === "object" && Object.keys(obj).length > 0;
  };

  $scope.hostClass = function(h) {
    var date = new Date(h.seenAt);
    var diff = +new Date() - date;
    var mins = diff / 1000 / 60;
    if (mins <= 10) {
      return "positive";
    } else if (mins <= 30) {
      return "warning";
    }
    return "negative";
  };
});
