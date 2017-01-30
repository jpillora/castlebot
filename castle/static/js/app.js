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
