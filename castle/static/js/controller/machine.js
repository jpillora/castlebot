module.controller("MachineController", function($scope, $http, $timeout) {
  var machine = $scope.machine = window.machine = this;
  machine.data = {};
  machine.status = {};
  $scope.$watch(
    "app.data.modules.machine",
    function(data) {
      machine.data = data || {};
      machine.status = machine.data.status || {};
    },
    true
  );

  $scope.round = function(n) {
    return Math.round(n * 10) / 10;
  };

  $scope.perc = function(n) {
    return $scope.round(100 * n);
  };
});
