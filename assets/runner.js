const src = location.pathname.match(/\/runner(\/[-\w]+=*)?/)?.[1]
src && (editor.value = atob(src.slice(1).replaceAll(/-/g, "+").replaceAll(/_/g, "/")))
editor.addEventListener("input", refreshURL)
refreshURL()

function refreshURL() {
	urlPane.url =
		"/run/" + btoa(editor.value).replaceAll(/\+/g, "-").replaceAll(/\//g, "_")
}
