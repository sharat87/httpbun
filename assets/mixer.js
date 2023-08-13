const urlEl = document.getElementById("url")
const formEl = document.getElementById("form")
const addBtnsEl = document.getElementById("addBtns")
const urlMismatchMessageEl = document.getElementById("urlMismatchMessage")

document.body.insertAdjacentHTML("afterbegin", `<style>
form {
	margin: 2.6em 0;
}
form p {
	display: flex;
	gap: 1em;
	align-items: start;
}
form p > [data-directive] {
	flex: 0 0 9em;
}
form p :is(input, textarea) {
	flex: 1;
}
input, button {
	font: inherit;
}
</style>`)

addBtnsEl.addEventListener("click", (event) => {
	if (event.target.tagName !== "BUTTON") {
		return
	}

	const directive = event.target.dataset.directive
	const id = "d" + formEl.children.length
	const parts = ["<p>", `<label for=${id} data-directive=${directive}>${event.target.innerText}</label>`]

	if (directive === "b64") {
		parts.push(`<textarea id=${id} placeholder='Plain text response body' rows=4></textarea>`)

	} else if (directive === "s") {
		parts.push(`<input id=${id} type=number min=100 max=599 required value=200>`)

	} else if (directive === "h") {
		parts.push(`<input id=${id} required pattern='^[-_a-zA-Z0-9]+$' placeholder=key>:<input placeholder=value>`)

	} else if (directive === "c") {
		parts.push(`<input id=${id} required placeholder=name>:<input placeholder=value>`)

	} else if (directive === "cd") {
		parts.push(`<input id=${id} required placeholder=name>`)

	} else if (directive === "r") {
		parts.push(`<input id=${id} type=url placeholder=URL>`)

	} else if (directive === "d") {
		parts.push(`<input id=${id} type=number required placeholder='Delay seconds' min=0 max=10 step=.1>`)

	}

	parts.push("<button>Del</button></p>")
	formEl.insertAdjacentHTML("beforeend", parts.join(""))
	formEl.lastElementChild.querySelector("input, textarea").focus()
	recomputeURL()
	checkAddButtons()
})

formEl.addEventListener("click", (event) => {
	if (event.target.tagName === "BUTTON" && event.target.innerText === "Del") {
		event.target.closest("p").remove()
		recomputeURL()
		checkAddButtons()
	}
})

formEl.addEventListener("keydown", () => setTimeout(recomputeURL))
formEl.addEventListener("keyup", recomputeURL)  // for paste
formEl.addEventListener("change", recomputeURL)

urlEl.closest("p").addEventListener("click", () => {
	navigator.clipboard.writeText(urlEl.innerText)
	showGhost(urlEl)
})

loadFromURL()

function loadFromURL() {
	if (URL_MIX_ENTRIES == null) {
		addBtnsEl.children[0].click()
		formEl.querySelector("input, textarea").focus()
		return
	}

	for (const entry of URL_MIX_ENTRIES) {
		document.querySelector("button[data-directive=" + entry.dir + "]").click()
		const part = formEl.lastElementChild
		const inputs = part.querySelectorAll("input, textarea")
		for (let i = 0; i < inputs.length; i++) {
			const data = entry.args[i] ?? ""
			inputs[i].value = entry.dir === "b64" ? atob(data) : data
		}
	}

	recomputeURL()
	checkAddButtons()
}

function recomputeURL() {
	const url = new URL(location.href)
	url.search = url.hash = ""
	const currentURLPath = url.pathname
	const pathItems = ["/mix"]

	for (const p of formEl.querySelectorAll("p")) {
		const directive = p.firstElementChild.dataset.directive
		const values = Array.from(p.querySelectorAll("input, textarea")).map((el) => el.value)
		pathItems.push("/" + directive + "=")

		if (directive === "b64") {
			pathItems.push(btoa(values[0]))

		} else if (directive === "s") {
			pathItems.push(values[0])

		} else if (directive === "h") {
			pathItems.push(values[0] + ":" + values[1])

		} else if (directive === "c") {
			pathItems.push(values[0] + ":" + values[1])

		} else if (directive === "cd") {
			pathItems.push(values[0])

		} else if (directive === "r") {
			pathItems.push(encodeURIComponent(values[0]))

		} else if (directive === "d") {
			pathItems.push(values[0])

		}
	}

	url.pathname = pathItems.join("")
	urlEl.innerText = url.toString()

	urlMismatchMessageEl.style.display =
		currentURLPath !== "/mixer" && url.pathname !== currentURLPath.replace("/mixer", "/mix") ? "" : "none"
}

function checkAddButtons() {
	const addedLabels = new Set
	const unrepeatableLabels = new Set

	for (const btn of addBtnsEl.querySelectorAll("button[no-repeat]")) {
		unrepeatableLabels.add(btn.innerText)
	}

	for (const p of formEl.querySelectorAll("p > [data-directive]")) {
		addedLabels.add(p.innerText)
	}

	for (const btn of addBtnsEl.querySelectorAll("button")) {
		btn.disabled = addedLabels.has(btn.innerText) && unrepeatableLabels.has(btn.innerText);
	}
}

function showGhost(el) {
	const rect = el.getBoundingClientRect()
	const ghost = document.createElement("div")
	ghost.innerText = "Copied!"
	ghost.className = "ghost"
	ghost.style.left = rect.x + "px"
	ghost.style.top = rect.y + "px"
	ghost.style.minWidth = rect.width + "px"
	ghost.style.height = rect.height + "px"
	ghost.addEventListener("animationend", ghost.remove.bind(ghost))
	document.body.appendChild(ghost)
}
