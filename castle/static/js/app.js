console.log(42);

var data = {};
var conn = velox("/sync", data);
conn.onupdate = function() {
  console.log(data);
};
