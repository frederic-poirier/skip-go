const inputs = document.querySelectorAll("fieldset input");

inputs.forEach((input, index) => {
    input.addEventListener("beforeinput", (e) => {
        if (e.inputType == "deleteContentBackward") {
            if (index == 0) {
                return;
            }

            e.preventDefault();
            input.value = "";
            const previous = inputs[index - 1];
            previous.focus();
        } else if (e.inputType == "insertText") {
            if (index == inputs.length - 1) {
                return;
            }

            e.preventDefault();
            input.value = e.data;
            const next = inputs[index + 1];
            next.focus();
        }
    });
});
