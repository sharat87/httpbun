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
		this.className = "entry"

		this.innerHTML = "<div>" + this.formControl() + "</div>"

		const label = el(`<label style="flex:0 0 min(20%, 9em)">${LABELS[this.directive]}</label>`)
		this.insertBefore(label, this.firstChild)

		const delBtn = el(`<button class=del>&times;</button>`)
		delBtn.addEventListener("click", this.remove.bind(this))
		this.append(delBtn)

		const input = this.querySelector("input, textarea")
		label.setAttribute("for", input.id = "e" + Math.random())
		input.focus()
	}

	formControl() {
		throw new Error("formControl unimplemented")
	}

	get path() {
		return this.directive ? "/" + this.directive + "=" + this.value : ""
	}
}

customElements.define("entry-s", class extends EntryElement {
	formControl() {
		return [
			`<input placeholder='One or more status codes, comma separated' required value=200>`,
			"<small>A single status integer, like <code>200</code> or <code>400</code>, or a comma-separated list of",
			" statuses, like <code>200, 400, 401</code>, out of which a random one will be used.</small>",
		].join("")
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
		return `<div class=flex><input required pattern='^[-_a-zA-Z0-9]+$' placeholder=key>:<input placeholder=value></div>`
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
		return `<div class=flex><input required placeholder=name>:<input placeholder=value></div>`
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

customElements.define("entry-slack", class extends EntryElement {
	formControl() {
		return [
			`<input pattern="^https://hooks.slack.com/services/[/\w]+" required placeholder="Slack webhook URL">`,
			`<small>Full request details will be posted as a message to this Slack webhook.</small>`,
		].join("")
	}

	get value() {
		return encodeURIComponent(this.querySelector("input").value.match(/services\/(.+)$/)[1])
	}

	set value(v) {
		this.querySelector("input").value = "https://hooks.slack.com/services/" + decodeURIComponent(v)
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
	const parts = Array.from(location.pathname.matchAll(/\/\w+=[^\/]+/g))
	for (const [part] of parts) {
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
	const pathItems = ["/mix"]

	for (const p of formEl.querySelectorAll("p, .entry")) {
		pathItems.push(p.path)
	}

	urlPane.path = pathItems.join("")
}

function checkAddButtons() {
	const added = new Set(Array.from(formEl.children).map(e => e.directive))
	for (const btn of addBtnsEl.querySelectorAll("button[no-repeat]")) {
		btn.disabled = added.has(btn.dataset.directive)
	}
}
