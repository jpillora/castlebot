module.controller("ServerController", function($scope, $http, $timeout) {
  var s = $scope.s = this;
  s.save = function() {
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
