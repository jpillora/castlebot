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

module.directive("since", function(sinceMillis) {
  return {
    restrict: "A",
    link: function(s, e, attrs) {
      var d, t;
      var check = function() {
        clearTimeout(t);
        if (d && !isNaN(d) && d instanceof Date) {
          var millis = +new Date() - d;
          e.text(sinceMillis(millis) + ("ago" in attrs ? " ago" : ""));
          if (millis < 60 * 1000) {
            t = setTimeout(check, 1000);
          } else if (millis < 60 * 60 * 1000) {
            t = setTimeout(check, 60 * 1000);
          }
        }
      };
      s.$watch(attrs.since, function(s) {
        d = new Date(s);
        e.attr("title", d.toString());
        check();
      });
    }
  };
});

module.directive("ezHttp", function($timeout) {
  //ez-http directive
  return {
    restrict: "A",
    link: function(scope, elem, attrs) {
      //extract function and args from attribute
      if (!/(\w+)\(([^\)]*)\)/.test(attrs.ezHttp))
        return alert("invalid ez-http string: " + attrs.ezHttp);
      var fnExpr = RegExp.$1;
      var argsExpr = "[" + RegExp.$2 + "]";
      var fn = scope.$eval(fnExpr);
      if (!fn || typeof fn !== "function")
        return alert("invalid ez-http fn: " + attrs.ezHttp);
      //find button
      var btn = null;
      var form = null;
      switch (elem[0].tagName.toLowerCase()) {
        case "button":
          btn = elem;
          break;
        case "form":
          form = elem;
          btn = form.find("button[type=submit]");
          if (btn.length !== 1)
            return alert("ez-http form btn not found: " + attrs.ezHttp);
          break;
        default:
          return alert("invalid ez-http elem type: " + attrs.ezHttp);
      }
      //elems
      var error = angular.element("<span class='error-msg'/>");
      var success = angular.element("<span class='success-msg'/>");
      var children = {
        user: Array.prototype.slice.call(btn[0].childNodes),
        success: [success[0]],
        error: [error[0]]
      };
      var show = function(id) {
        var parent = btn[0];
        for (var k in children) {
          var add = id === k;
          var elems = children[k];
          elems.forEach(function(e) {
            var has = e.parentNode === parent;
            if (has && !add || !has && add)
              parent[(add ? "append" : "remove") + "Child"](e);
          });
        }
      };
      var disabledExpr = attrs.ngDisabled;
      var disable = function(bool) {
        //disabled by expr?
        if (!bool && !!scope.$eval(disabledExpr)) bool = true;
        btn.attr("disabled", bool ? "disabled" : null);
      };
      //options
      var loadingClass = attrs.ezLoadingClass || "loading";
      var errorClass = attrs.ezErrorClass || "red";
      var errorMsg = attrs.ezErrorMsg || "Error";
      var successClass = attrs.ezSuccessClass || "green";
      var successMsg = attrs.ezSuccessMsg || "Success";
      var confirm = "ezConfirm" in attrs;
      var confirmTimer = null;
      //submit fn
      var submit = function() {
        //always apply
        $timeout(function() {
          scope.$apply();
        });
        //eval args
        var args = scope.$eval(argsExpr) || [];
        //confirm? flag confirm on first submit
        if (confirm) {
          $timeout.cancel(confirmTimer);
          if (fn.confirm) {
            fn.confirm = false;
          } else {
            fn.confirm = true;
            confirmTimer = $timeout(
              function() {
                fn.confirm = false;
              },
              5000
            );
            return;
          }
        }
        //fn!
        var p = fn.apply(scope, args);
        if (p && typeof p === "object" && typeof p.then === "function") {
          handlePromise(p);
        }
        return p;
      };
      //reset element
      var resetTimer;
      var reset = function() {
        btn.removeClass(successClass);
        btn.removeClass(errorClass);
        show("user");
      };
      var finaly = function() {
        btn.removeClass(loadingClass);
        fn.ing = false;
        disable(false);
      };
      //handle loading, errors/success promise
      var handlePromise = function(p) {
        fn.ing = true;
        fn.error = null;
        fn.success = null;
        clearTimeout(resetTimer);
        btn.addClass(loadingClass);
        disable(true);
        //can only rely on then(), manually call finaly
        p.then(
          function(resp) {
            //promise resolved!
            var msg = resp && resp.data;
            if (typeof msg !== "string" || msg === "") msg = successMsg;
            fn.success = msg;
            btn.addClass(successClass);
            success.text(msg);
            show("success");
            finaly();
            resetTimer = setTimeout(reset, 2000);
          },
          function(resp) {
            //promise rejected!
            var msg = resp.error || resp.data;
            if (msg.error) msg = msg.error;
            if (typeof msg !== "string" || msg === "") msg = errorMsg;
            fn.error = msg;
            btn.addClass(errorClass);
            btn.addClass("buzz-out");
            setTimeout(btn.removeClass.bind(btn, "buzz-out"), 2000);
            error.text(msg);
            show("error");
            finaly();
            resetTimer = setTimeout(reset, 5000);
          }
        );
      };
      //standard ng-click evaluate
      if (form) form.on("submit", submit);
      else if (btn) btn.on("click", submit);
    }
  };
});
