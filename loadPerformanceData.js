$(document).ready(function () {
var arr = window.performance.getEntriesByType("resource")
jsonObj = [];
console.log(jsonObj);
  $.each( arr, function( i, val ) {
      var name = val.name;
      var entryType = val.entryType;
      var startTime = val.fetchStart;
      var endTime = val.duration;
      var initiatorType = val.initiatorType;

      item = {}
      item ["name"] = name;
      item ["entryType"] = entryType;
      item ["startTime"] = startTime;
      item ["endTime"] = endTime;
      item ["initiatorType"] = initiatorType;

      jsonObj.push(item);
   });
   jsonString = JSON.stringify(jsonObj);
   console.log(jsonString);
   $.ajax({
       type: "POST",
       url: "http://localhost:8699/endpoint",
       data: jsonString
       // success: success,
       // dataType: dataType
     });
});
