module.controller("AuthController", function($scope, $http, $timeout) {
  var auth = (window.auth = $scope.auth = this);
  auth.settings = {};

  //webcam enabled? start/stop
  $scope.$watch(
    "data.modules.auth.settings",
    function(settings) {
      auth.settings = angular.copy(settings || {});
    },
    true
  );

  auth.update = function() {
    var data = angular.extend({}, auth.settings);
    $http({url: "m/auth/settings", method: "PUT", data: data}).then(
      function(resp) {
        console.info("succeses", resp.data);
      },
      function(resp) {
        console.warn(resp.data);
      }
    );
  };
});
