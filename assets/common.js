function el(html) {
	const nodes = new DOMParser().parseFromString(html, "text/html").body.children
	if (nodes.length > 1) {
		const f= document.createDocumentFragment()
		f.append(...nodes)
		return f
	} else return nodes[0]
}

customElements.define("url-pane", class extends HTMLElement {
	constructor() {
		super()
		this.url = ""
	}

	connectedCallback() {
		this.a = this.querySelector("a")

		const onClick = ({currentTarget: e}) => {
			navigator.clipboard.writeText(e.dataset.format.replace("%", this.url))
			showGhost(e)
		}

		this.querySelectorAll("button[data-format]")
			.forEach(b => b.addEventListener("click", onClick))
	}

	set url(value) {
		this.a && (this.a.href = this.a.innerText = value)
	}

	get url() {
		return this.a.href
	}
})
