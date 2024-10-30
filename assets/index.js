const ID = {
	n: 0,
	inc: () => ++ID.n,
}

customElements.define("try-it-out", class extends HTMLElement {
	constructor() {
		super()
		this.form = null
	}

	connectedCallback() {
		const dd = this.closest("dd")
		const dts = []
		const params = new Set
		let e = dd.previousElementSibling
		while (e?.tagName === "DT") {
			dts.push(e)
			for (const m of e.innerText.matchAll(/\{(\w+)}/g)) {
				params.add(m[1])
			}
			e = e.previousElementSibling
		}

		this.innerHTML = [
			"<details><summary>Try it out</summary>",
			"<form><div class=grid>",
			`<label for=a${ID.inc()}>Method:</label><input id=a${ID.n} name=method value=GET>`,
			[...params].map(p => `<label for=a${ID.inc()}><code>${p}</code>:</label><input id=a${ID.n} name="${p}">`).join(""),
			"</div><p class=flex>",
			(dts.map(e => `<button name="url" value="${e.innerText}">Send <code>${e.innerText}</code></button>`)).reverse().join(""),
			"</p>",
			"<p>Use the Network tab from your browser to see response and to copy request to <code>curl</code>.</p>",
			"</form>",
			"</details>",
		].join("")

		;(this.form = this.querySelector("form")).addEventListener("submit", this.send.bind(this))
	}

	send(event) {
		event.preventDefault()

		const url = event.submitter.value.replace(/^\//, "")
			.replaceAll(/\{(\w+)}/g, (_, param) => this.form[param].value)

		fetch(url, {
			method: this.form.method.value || "GET",
		})
	}
})

document.body.querySelectorAll(":scope > dl > dd")
	.forEach(e => e.previousElementSibling?.innerText.startsWith("/")
		&& e.insertAdjacentHTML("beforeend", "<try-it-out></try-it-out>"))
