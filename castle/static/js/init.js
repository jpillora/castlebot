window.module = angular.module("castle", []);
window.module.config(function($httpProvider) {
  $httpProvider.defaults.withCredentials = true;
});
