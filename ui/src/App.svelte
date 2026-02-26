<script lang="ts">
  import { onMount } from 'svelte';
  import Projects from '$lib/Projects.svelte';
  import Terminal from '$lib/Terminal.svelte';
  import { Cast as CastIcon, TerminalSquare } from 'lucide-svelte';

  let currentView = 'projects';
  let sshHost = '';
  let projectId = '';

  function openSSH(event: CustomEvent<{ host: string, projectId: string }>) {
    sshHost = event.detail.host;
    projectId = event.detail.projectId;
    currentView = 'terminal';
  }

  function goBack() {
    currentView = 'projects';
    sshHost = '';
  }
</script>

<main class="min-h-screen flex flex-col bg-slate-950 text-slate-50">
  <header class="bg-slate-900 border-b border-slate-800 p-4 flex justify-between items-center shadow-md">
    <div class="flex items-center gap-3">
      <div class="w-10 h-10 bg-indigo-600 rounded-lg flex items-center justify-center font-bold shadow-lg shadow-indigo-600/20">
        <CastIcon class="w-6 h-6 text-white" />
      </div>
      <h1 class="text-2xl font-bold tracking-tight">Cast</h1>
    </div>
    {#if currentView === 'terminal'}
      <button 
        class="px-4 py-2 bg-slate-800 hover:bg-slate-700 border border-slate-700 rounded-md text-sm font-medium transition-all shadow-sm"
        onclick={goBack}
      >
        Back to Projects
      </button>
    {/if}
  </header>

  <div class="flex-1 overflow-hidden flex flex-col">
    {#if currentView === 'projects'}
      <Projects on:ssh={openSSH} />
    {:else if currentView === 'terminal'}
      <div class="flex-1 flex flex-col p-6 max-w-[1600px] w-full mx-auto">
        <div class="flex items-center gap-3 mb-4">
          <TerminalSquare class="w-5 h-5 text-indigo-400" />
          <h2 class="text-lg font-mono font-semibold">
            ssh <span class="text-indigo-300">{sshHost}</span>
          </h2>
        </div>
        <div class="flex-1 border border-slate-800 rounded-xl overflow-hidden shadow-2xl bg-slate-950">
          <Terminal {projectId} {sshHost} />
        </div>
      </div>
    {/if}
  </div>
</main>
