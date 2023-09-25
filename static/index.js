document.addEventListener("keydown", function (event) {
    document.getElementById("hide").value = event.shiftKey ? "true" : "false";
});
document.addEventListener("keyup", function (event) {
    document.getElementById("hide").value = "false";
});

function addCol() {
    const table = document.getElementById("table");
    const nCustomCols = document.getElementsByClassName("custom-col-th").length;
    const newTH = document.createElement("th");
    newTH.className = "custom-col-th";
    newTH.innerHTML = `
        <input name="column-name-${nCustomCols}"
               hx-post="/set-column-name"
               hx-swap="none" />
    `;
    htmx.process(newTH);
    document.getElementById("add-column-cell").before(newTH);
    table.querySelectorAll(".body-row").forEach(function (elem, index) {
        const newTD = document.createElement("td");
        newTD.innerHTML = `
            <input name="custom-cell-${nCustomCols},${index}"
                   hx-post="/set-cell"
                   hx-swap="none" />
        `;
        htmx.process(newTD)
        elem.append(newTD);
    })
}
