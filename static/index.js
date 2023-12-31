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
document.addEventListener("focusout", function (event) {
    let target = event.target;
    if (target.classList.contains("extra-cell-formula")) {
        target = target.parentElement;
    }
    if (target.classList.contains("extra-cell")) {
        target.querySelector("span").classList.remove("hide");
        target.querySelector("input").classList.add("hide");
    }
});
document.addEventListener("click", function (event) {
    const modals = Array.from(document.getElementsByClassName("modal"));
    modals.forEach(function (elem) {
        elem.remove();
    });
});
