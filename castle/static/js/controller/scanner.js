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
});
