document.addEventListener("keydown", function (event) {
    document.getElementById("hide").value = event.shiftKey ? "true" : "false";
});
document.addEventListener("keyup", function (event) {
    document.getElementById("hide").value = "false";
});
