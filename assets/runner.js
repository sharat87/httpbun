const src = location.pathname.match(/\/runner(\/[-\w]+=*)?/)?.[1]
src && (editor.value = base64ToUtf8(src.slice(1).replaceAll(/-/g, "+").replaceAll(/_/g, "/")))
editor.addEventListener("input", refreshURL)
refreshURL()

function refreshURL() {
	urlPane.path =
		"/run/" + utf8ToBase64(editor.value).replaceAll(/\+/g, "-").replaceAll(/\//g, "_")
}

// Unlike plain btoa/atob, these handle any Unicode text, by encoding it as UTF-8.
function utf8ToBase64(text) {
	return btoa(Array.from(new TextEncoder().encode(text), (b) => String.fromCharCode(b)).join(""))
}

function base64ToUtf8(data) {
	return new TextDecoder().decode(Uint8Array.from(atob(data), (c) => c.charCodeAt(0)))
}
