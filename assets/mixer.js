const LABELS = {}
for (const b of addBtnsEl.querySelectorAll("button")) {
	LABELS[b.dataset.directive] = b.textContent
}

class EntryElement extends HTMLElement {
	constructor() {
		super()
		this.directive = ""
	}

	connectedCallback() {
		this.directive = this.tagName.substring("entry-".length).toLowerCase()
		this.className = "entry flex"

		this.innerHTML = this.formControl()

		const label = el(`<label style="flex:0 0 min(20%, 9em);text-align:right">${LABELS[this.directive]}</label>`)
		this.insertBefore(label, this.firstChild)

		const delBtn = el(`<button class=del>&times;</button>`)
		delBtn.addEventListener("click", this.remove.bind(this))
		this.append(delBtn)

		const input = this.querySelector("input, textarea")
		label.setAttribute("for", input.id = "e" + Math.random())
		input.focus()
	}

	get path() {
		return this.directive ? "/" + this.directive + "=" + this.value : ""
	}
}

customElements.define("entry-s", class extends EntryElement {
	formControl() {
		return `<input placeholder='One or more status codes, comma separated' required value=200>`
	}

	get value() {
		return this.querySelector("input").value
	}

	set value(v) {
		this.querySelector("input").value = v
	}
})

customElements.define("entry-h", class extends EntryElement {
	formControl() {
		return `<input required pattern='^[-_a-zA-Z0-9]+$' placeholder=key>:<input placeholder=value>`
	}

	get value() {
		const inputs = this.querySelectorAll("input")
		return inputs[0].value + ":" + encodeURIComponent(inputs[1].value)
	}

	set value(v) {
		const [_, name, value] = v.match(/^(.+?):(.+)$/)
		const inputs = this.querySelectorAll("input")
		inputs[0].value = name
		inputs[1].value = decodeURIComponent(value)
	}
})

customElements.define("entry-c", class extends EntryElement {
	formControl() {
		return `<input required placeholder=name>:<input placeholder=value>`
	}

	get value() {
		const inputs = this.querySelectorAll("input")
		return inputs[0].value + ":" + encodeURIComponent(inputs[1].value)
	}

	set value(v) {
		const [_, name, value] = v.match(/^(.+?):(.+)$/)
		const inputs = this.querySelectorAll("input")
		inputs[0].value = name
		inputs[1].value = decodeURIComponent(value)
	}
})

customElements.define("entry-cd", class extends EntryElement {
	formControl() {
		return `<input required placeholder=name>`
	}

	get value() {
		return this.querySelector("input").value
	}

	set value(v) {
		this.querySelector("input").value = v
	}
})

customElements.define("entry-b64", class extends EntryElement {
	formControl() {
		return `<textarea placeholder='Plain text response body' rows=4></textarea>`
	}

	get value() {
		return btoa(this.querySelector("textarea").value)
	}

	set value(v) {
		this.querySelector("textarea").value = atob(v)
	}
})

customElements.define("entry-t", class extends EntryElement {
	formControl() {
		return `<textarea placeholder='Golang template response body' rows=4></textarea>`
	}

	get value() {
		return btoa(this.querySelector("textarea").value)
	}

	set value(v) {
		this.querySelector("textarea").value = atob(v)
	}
})

customElements.define("entry-r", class extends EntryElement {
	formControl() {
		return `<input type=url placeholder=URL>`
	}

	get value() {
		return encodeURIComponent(this.querySelector("input").value)
	}

	set value(v) {
		this.querySelector("input").value = decodeURIComponent(v)
	}
})

customElements.define("entry-d", class extends EntryElement {
	formControl() {
		return `<input type=number required placeholder='Delay seconds' min=0 max=10 step=.1>`
	}

	get value() {
		return this.querySelector("input").value
	}

	set value(v) {
		this.querySelector("input").value = v
	}
})

addBtnsEl.addEventListener("click", (event) => {
	if (event.target.dataset.directive)
		formEl.append(document.createElement("entry-" + event.target.dataset.directive))
})

formEl.addEventListener("input", recomputeURL)
formEl.addEventListener("change", recomputeURL)

new MutationObserver(() => {
	recomputeURL()
	checkAddButtons()
}).observe(formEl, { childList: true })

loadFromURL()

function loadFromURL() {
	const [_, ...parts] = location.pathname.match(/\/mixer(\/\w+=[^\/]+)+/) ?? []
	for (const part of parts) {
		const [_, dir, data] = part.match(/^\/(\w+)=(.*)$/)
		addBtnsEl.querySelector("button[data-directive=" + dir + "]").click()
		formEl.lastElementChild.value = data
	}
	if (parts.length === 0) {
		addBtnsEl.querySelector("button").click()
	}
}

function recomputeURL() {
	const url = new URL(location.href)
	url.search = url.hash = ""
	const currentURLPath = url.pathname
	const pathItems = ["/mix"]

	for (const p of formEl.querySelectorAll("p, .entry")) {
		pathItems.push(p.path)
	}

	url.pathname = pathItems.join("")
	urlPane.url = url.toString()

	urlMismatchMessageEl.style.display =
		currentURLPath !== "/mixer" && url.pathname !== currentURLPath.replace("/mixer", "/mix") ? "" : "none"
}

function checkAddButtons() {
	const added = new Set(Array.from(formEl.children).map(e => e.directive))
	for (const btn of addBtnsEl.querySelectorAll("button[no-repeat]")) {
		btn.disabled = added.has(btn.dataset.directive)
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
