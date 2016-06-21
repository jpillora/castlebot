var data = {};
var conn = velox("/sync", data);
conn.onupdate = function() {
  preview.innerHTML = JSON.stringify(data, null, 2);
};
