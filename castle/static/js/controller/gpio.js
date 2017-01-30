module.controller("GPIOController", function($scope, $http, $timeout) {
  var gb = window.gb = $scope.gb = this;
  gb.toggle = function() {
    if (gb.toggling || gb.toggled) return;
    gb.error = null;
    gb.toggling = true;
    $http({method: "GET", url: "/gpio", params: {p: 23, d: "1000ms"}})
      .then(
        function() {
          gb.toggled = true;
          $timeout(
            function() {
              gb.toggled = false;
            },
            3000
          );
        },
        function(resp) {
          gb.error = resp.data;
        }
      )
      .finally(function() {
        gb.toggling = false;
      });
  };
});
