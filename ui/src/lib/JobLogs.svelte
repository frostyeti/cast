<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import AnsiToHtml from 'ansi-to-html';
  
  export let runId: string;
  
  let logsContainer: HTMLElement;
  let eventSource: EventSource;
  let htmlLogs = '';
  const converter = new AnsiToHtml({
    escapeXML: true,
    newline: false // We use whitespace-pre-wrap, so \n is rendered naturally
  });

  function scrollToBottom() {
    if (logsContainer) {
      logsContainer.scrollTop = logsContainer.scrollHeight;
    }
  }

  onMount(() => {
    let url = `/api/v1/streams/${runId}`;
    if (import.meta.env.DEV) {
      url = `http://localhost:3000${url}`;
    }

    eventSource = new EventSource(url);

    eventSource.onmessage = (event) => {
      htmlLogs += converter.toHtml(event.data);
      // Use requestAnimationFrame or setTimeout to wait for DOM update
      requestAnimationFrame(scrollToBottom);
    };

    eventSource.onerror = (err) => {
      console.error('SSE Error:', err);
      // Close on error/finish
      eventSource.close();
      htmlLogs += '\n<span style="color:#aaa">[Stream closed]</span>\n';
      requestAnimationFrame(scrollToBottom);
    };
  });

  onDestroy(() => {
    if (eventSource) {
      eventSource.close();
    }
  });
</script>

<div 
  class="h-full w-full overflow-y-auto bg-black p-3 font-mono text-xs text-slate-300 whitespace-pre-wrap"
  bind:this={logsContainer}
>
  {@html htmlLogs}
</div>
