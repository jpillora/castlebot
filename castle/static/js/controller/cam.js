module.controller("CamController", function($scope, $http) {
  var cam = window.cam = $scope.cam = this;
  cam.started = false;
  cam.title = "Gate";
  cam.range = "hour";
  cam.timestamp = "";
  cam.blob = null;
  cam.liveIndex = 0;
  cam.timeIndex = "1000";
  cam.viewMode = "raw";

  var update = cam.update = function(obj) {
    var data = angular.extend($scope.app.data.settings.webcam, obj || {});
    $http({url: "/settings/webcam", method: "PUT", data: data}).then(function(
      resp
    ) {
      console.info("succeses", resp.data);
    }, function(resp) {
      console.warn(resp.data);
    });
  };

  //webcam enabled? start/stop
  $scope.$watch(
    "data.webcam.settings",
    function(settings) {
      if (settings && settings.webcam && settings.webcam.enabled) {
        cam.started = true;
        refresh();
      } else {
        cam.started = false;
        clearTimeout(refresh.t);
      }
    },
    true
  );

  var refreshError = function(err) {
    console.log("update error", err);
    setTimeout(refresh, 3000);
  };

  var refresh = cam.refresh = function(isodate) {
    clearTimeout(refresh.t);
    if (!cam.started) {
      return;
    }
    var url;
    //realtime
    var realtime = cam.timeIndex === "1000";
    if (isodate) {
      url = "webcam/snap/" + isodate;
    } else {
      url = "webcam/live/" + cam.liveIndex + "/" + cam.viewMode;
    }
    fetch(url).then(
      function(resp) {
        //refresh next interval
        var interval = parseInt(resp.headers.get("Interval")) * 1000;
        //get date
        var ts = moment(new Date(resp.headers.get("Last-Modified")));
        if (!ts.isValid()) {
          cam.timestamp = "-";
        } else {
          var now = moment();
          if (now.diff(ts, "second") == 0) {
            cam.timestamp = now.diff(ts) + " ms ago";
          } else if (now.diff(ts, "minute") == 0) {
            var s = now.diff(ts, "second");
            cam.timestamp = s + " second" + (s == 1 ? "" : "s") + " ago";
          } else if (now.diff(ts, "hour") == 0) {
            var m = now.diff(ts, "minute");
            cam.timestamp = m + " minute" + (m == 1 ? "" : "s") + " ago";
          } else {
            cam.timestamp = ts.format("h:mma");
            if (now.diff(ts, "day") > 0) {
              cam.timestamp += ts.format(" DD/MM/YYYY");
            }
          }
        }
        //refresh next interval
        cam.nextSnap = resp.headers.get("Next");
        cam.prevSnap = resp.headers.get("Prev");
        //load data
        resp.blob().then(
          function(blob) {
            cam.blob = blob;
            $scope.$apply();
            if (!isodate && !isNaN(interval)) {
              clearTimeout(refresh.t);
              refresh.t = setTimeout(refresh, interval);
            }
          },
          refreshError
        );
      },
      refreshError
    );
  };
  //refresh on slider change
  var rangeChange = cam.rangeChange = function() {
    var isodate = null;
    if (cam.timeIndex === "1000") {
      cam.timeSlider = null;
    } else {
      var now = moment();
      var days = 1;
      var past = moment();
      switch (cam.range) {
        case "month":
          past.subtract(31, "day");
          break;
        case "week":
          past.subtract(7, "day");
          break;
        case "day":
          past.subtract(1, "day");
          break;
        case "hour":
          past.subtract(1, "hour");
          break;
      }
      var duration = now.diff(past);
      var percent = parseInt(cam.timeIndex) / 10;
      if (isNaN(percent)) return;
      var factor = 1 - percent / 100;
      var target = now.subtract(duration * factor);
      var isodate = target.toISOString().replace(/\.\d+Z$/, "Z");
      cam.timeSlider = isodate;
    }
    clearTimeout(rangeChange.t);
    rangeChange.t = setTimeout(refresh.bind(null, isodate), 250);
  };
  $scope.$watch("cam.range", rangeChange);
  $scope.$watch("cam.timeIndex", rangeChange);
});
