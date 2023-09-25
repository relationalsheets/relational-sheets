document.addEventListener("keydown", function (event) {
    document.getElementById("hide").value = event.shiftKey ? "true" : "false";
});
document.addEventListener("keyup", function (event) {
    document.getElementById("hide").value = "false";
});

function addCol() {
    document.getElementById("add-col-button").remove();
    const table = document.getElementById("table");
    const nCustomCols = document.getElementsByClassName("custom-col-th").length;
    const newTH = document.createElement("th");
    newTH.className = "custom-col-th";
    newTH.innerHTML = `
        <div class="flex">
            <input name="column-name-${nCustomCols}"
                   hx-post="/set-col-name"
                   hx-swap="none" />
            <button id="add-col-button"
                hx-post="/add-column"
                hx-swap="none"
                onclick="addCol()">
                + Col
            </button>
        </div>
    `;
    htmx.process(newTH);
    document.getElementById("header-row").append(newTH);
    table.querySelectorAll(".body-row").forEach(function (elem, index) {
        const newTD = document.createElement("td");
        newTD.className = "is-null"
        newTD.innerHTML = `
            <input name="custom-cell-${nCustomCols},${index}"
                   class="custom-cell-formula"
                   hx-post="/set-cell"
                   hx-swap="none" />
        `;
        htmx.process(newTD)
        elem.append(newTD);
    })
}
