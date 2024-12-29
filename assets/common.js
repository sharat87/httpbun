const pathPrefix = "{{.spec.PathPrefix}}"

customElements.define("url-pane", class extends HTMLElement {
	constructor() {
		super()
	}

	connectedCallback() {
		this.innerHTML = [
			`<a target="m" style="flex-basis:100%">https://httpbun.com/mix/...</a>`,
			`<button data-format="%">Copy URL</button>`,
			`<button data-format="curl '%'">Copy <code>curl</code></button>`,
			`<button data-format="http '%'">Copy <code>httpie</code></button>`,
		].join("")

		this.a = this.querySelector("a")
		this.className = "flex"
		this.style.margin = "2.6em 0"

		const onClick = ({currentTarget: e}) => {
			navigator.clipboard.writeText(e.dataset.format.replace("%", this.url))
			showGhost(e)
		}

		this.querySelectorAll("button[data-format]")
			.forEach(b => b.addEventListener("click", onClick))

		this.append(this.urlChangedMsg = el("<em style='display:none'>Note: Below URL is different from the one in the browser URL bar.</em>"))
		this.append(this.lengthWarning = el("<p style='display:none'>Note: URL longer than 2000 characters. May not work well with some HTTP clients.</p>"))
	}

	set path(path) {
		const url = `${location.protocol}//${location.host}${pathPrefix}${path}`
		this.a && (this.a.href = this.a.innerText = url)

		this.lengthWarning.style.display = url.length > 2000 ? "" : "none"

		const currentPath = popFirst(location.pathname.substring(pathPrefix.length))
		this.urlChangedMsg.style.display = currentPath && currentPath !== popFirst(path) ? "" : "none"
	}

	get url() {
		return this.a.href
	}
})

function el(html) {
	const nodes = new DOMParser().parseFromString(html, "text/html").body.children
	if (nodes.length > 1) {
		const f = document.createDocumentFragment()
		f.append(...nodes)
		return f
	} else return nodes[0]
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

function popFirst(path) {
	return path.replace(/^\/\w+/, "")
}
