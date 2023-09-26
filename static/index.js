let shiftPressed = false;
document.addEventListener("keydown", function (event) {
    shiftPressed = event.shiftKey;
});
document.addEventListener("keyup", function (event) {
    shiftPressed = false;
});
