#!/usr/bin/env bash

assert-contains '/oauth/authorize' '
<h2>Errors</h2>
<ol>
<li>Missing required param &#34;redirect_uri&#34;.
<li>Missing required param &#34;response_type&#34;.
</ol>
'

assert-contains \
	'/oauth/authorize?client_id=123&redirect_uri=https%3A%2F%2Foauthdebugger.com%2Fdebug&scope=one%20two%20three&response_type=token%20code&response_mode=query&state=random-state&nonce=random-nonce' \
	'<input type=submit name=decision value=Approve '
