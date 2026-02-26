# Issue 15: Stream SSH I/O over WebSockets

## Description
Allow a web UI client to open an interactive SSH session to any server defined in the project's inventory list through the Cast server.

## Requirements
- Create a WebSocket endpoint (e.g., `/api/ssh/{host_alias}`).
- Utilize `golang.org/x/crypto/ssh` to establish a backend connection using the credentials stored in the `cast` inventory for the requested host.
- Request a PTY session on the SSH backend.
- Bridge the WebSocket binary stream to the backend PTY Stdin/Stdout/Stderr to provide a fully interactive terminal session for `xterm.js` in the browser.

---

