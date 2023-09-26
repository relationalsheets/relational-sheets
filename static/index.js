let shiftPressed = false;
document.addEventListener("keydown", function (event) {
    shiftPressed = event.shiftKey;
});
document.addEventListener("keyup", function (event) {
    shiftPressed = false;
});
document.addEventListener("mousedown", function (event) {
    let target = event.target;
    if (target.classList.contains("extra-cell-value")) {
        target = target.parentElement;
    }
    if (target.classList.contains("extra-cell")) {
        target.querySelector("span").classList.add("hide");
        let input = target.querySelector("input");
        input.classList.remove("hide");
        setTimeout(() => input.focus());
    }
});
