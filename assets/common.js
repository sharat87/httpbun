const pathPrefix = "{{.pathPrefix}}"

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
	}

	connectedCallback() {
		this.a = this.querySelector("a")

		const onClick = ({currentTarget: e}) => {
			navigator.clipboard.writeText(e.dataset.format.replace("%", this.url))
			showGhost(e)
		}

		this.querySelectorAll("button[data-format]")
			.forEach(b => b.addEventListener("click", onClick))

		this.append(this.urlChangedMsg = el("<em style='display:none'>Note: Below URL is different from the one in the browser URL bar.</em>"))
	}

	set url(path) {
		const url = `${location.protocol}//${location.host}/${pathPrefix}/${path}`
		this.a && (this.a.href = this.a.innerText = url)

		const currentPath = popFirst(location.pathname.substring(pathPrefix.length))
		this.urlChangedMsg.style.display = currentPath && currentPath !== popFirst(path) ? "" : "none"
	}

	get url() {
		return this.a.href
	}
})

function popFirst(path) {
	return path.replace(/^\/\w+/, "")
}
