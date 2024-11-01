editor.addEventListener("input", refreshURL)
refreshURL()

function refreshURL() {
	const url = new URL(location.href)
	url.search = url.hash = ""
	const currentURLPath = url.pathname
	const pathItems = ["/run/", btoa(editor.value)
		.replaceAll(/\+/g, "-")
		.replaceAll(/\//g, "_"),
	]

	url.pathname = pathItems.join("")
	urlPane.url = url.toString()

	// todo: show a message if this URL is different from the one represented in the address bar.
}
