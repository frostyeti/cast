<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { Terminal as XTerm } from '@xterm/xterm';
  import { FitAddon } from '@xterm/addon-fit';

  export let projectId: string;
  export let sshHost: string;

  let terminalContainer: HTMLElement;
  let term: XTerm;
  let fitAddon: FitAddon;
  let ws: WebSocket;
  let resizeObserver: ResizeObserver;

  onMount(() => {
    // Initialize xterm.js
    term = new XTerm({
      cursorBlink: true,
      theme: {
        background: '#0f172a',
        foreground: '#f8fafc',
      },
      fontFamily: 'Menlo, Monaco, "Courier New", monospace',
    });

    fitAddon = new FitAddon();
    term.loadAddon(fitAddon);
    term.open(terminalContainer);
    fitAddon.fit();

    // Handle resizing
    resizeObserver = new ResizeObserver(() => {
      if (fitAddon) {
        fitAddon.fit();
      }
    });
    resizeObserver.observe(terminalContainer);

    // Connect to WebSocket
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    let wsUrl = `${protocol}//${window.location.host}/api/v1/projects/${projectId}/ssh/${sshHost}`;
    
    // In dev mode (Vite), point to the Go backend
    if (import.meta.env.DEV) {
      wsUrl = `ws://localhost:3000/api/v1/projects/${projectId}/ssh/${sshHost}`;
    }

    ws = new WebSocket(wsUrl);

    ws.onopen = () => {
      term.writeln(`\x1b[32mConnected to ${sshHost}\x1b[0m\r\n`);
    };

    ws.onmessage = (event) => {
      // If it's a blob/buffer, read it. 
      if (event.data instanceof Blob) {
        const reader = new FileReader();
        reader.onload = () => {
          if (reader.result) {
            term.write(new Uint8Array(reader.result as ArrayBuffer));
          }
        };
        reader.readAsArrayBuffer(event.data);
      } else {
        term.write(event.data);
      }
    };

    ws.onclose = () => {
      term.writeln('\r\n\x1b[31mConnection closed\x1b[0m');
    };

    ws.onerror = (err) => {
      term.writeln(`\r\n\x1b[31mConnection error\x1b[0m`);
      console.error(err);
    };

    // Send keystrokes to WebSocket
    term.onData((data) => {
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(data);
      }
    });
  });

  onDestroy(() => {
    if (resizeObserver) resizeObserver.disconnect();
    if (ws) ws.close();
    if (term) term.dispose();
  });
</script>

<div class="h-full w-full bg-slate-900 p-2" bind:this={terminalContainer}></div>
