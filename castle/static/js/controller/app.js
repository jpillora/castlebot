module.controller("AppController", function($scope, $rootScope) {
  var app = (window.app = $scope.app = this);
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

  $rootScope.ui = {
    shown: {
      scanner: true,
      webcam: true
    }
  };
});
