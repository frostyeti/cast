<script lang="ts">
  import { onMount, createEventDispatcher } from 'svelte';
  import { Button } from "$lib/components/ui/button";
  import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "$lib/components/ui/card";
  import * as Dialog from "$lib/components/ui/dialog";
  import * as Table from "$lib/components/ui/table";
  import { Badge } from "$lib/components/ui/badge";
  import { ScrollArea } from "$lib/components/ui/scroll-area";
  import { Play, Terminal as TerminalIcon, Server, RefreshCw, XCircle, Layers, Clock, History } from "lucide-svelte";
  import JobLogs from './JobLogs.svelte';

  const dispatch = createEventDispatcher();

  let projects: any[] = [];
  let selectedProject: any = null;
  let activeLogs = new Map<string, string>(); // runId -> jobId/taskId
  let historyOpen = false;
  let historyTitle = '';
  let historyRuns: any[] = [];
  let pollInterval: any;

  onMount(() => {
    fetchProjects();
    
    pollInterval = setInterval(() => {
      if (selectedProject) {
        refreshSelectedProject();
      }
    }, 3000);

    return () => {
      if (pollInterval) clearInterval(pollInterval);
    };
  });

  async function refreshSelectedProject() {
    if (!selectedProject) return;
    try {
      const [jobsRes, tasksRes] = await Promise.all([
        fetch(`/api/v1/projects/${selectedProject.id}/jobs`),
        fetch(`/api/v1/projects/${selectedProject.id}/tasks`)
      ]);
      
      if (jobsRes.ok) selectedProject.jobs = await jobsRes.json();
      if (tasksRes.ok) selectedProject.tasks = await tasksRes.json();
      // Trigger reactivity
      selectedProject = selectedProject;
    } catch (e) {
      console.error('Failed to refresh project details', e);
    }
  }

  function formatDuration(start: string, end: string | null) {
    if (!end) return '-';
    const ms = new Date(end).getTime() - new Date(start).getTime();
    if (ms < 1000) return `${ms}ms`;
    return `${(ms / 1000).toFixed(1)}s`;
  }

  async function fetchHistory(type: string, id: string) {
    historyTitle = `${type.charAt(0).toUpperCase() + type.slice(1)}: ${id}`;
    historyOpen = true;
    historyRuns = [];
    try {
      const res = await fetch(`/api/v1/projects/${selectedProject.id}/${type}s/${id}/runs`);
      if (res.ok) {
        historyRuns = await res.json();
      }
    } catch (e) {
      console.error('Failed to fetch history', e);
    }
  }

  function viewLogs(runId: string, runName: string) {
    historyOpen = false;
    activeLogs.set(runId, runName);
    activeLogs = activeLogs;
  }

  async function fetchProjects() {
    try {
      const res = await fetch('/api/v1/projects');
      if (res.ok) {
        projects = await res.json();
      }
    } catch (e) {
      console.error('Failed to fetch projects', e);
    }
  }

  async function selectProject(p: any) {
    selectedProject = p;
    // Fetch full project details like jobs, tasks, inventory
    try {
      const [jobsRes, tasksRes] = await Promise.all([
        fetch(`/api/v1/projects/${p.id}/jobs`),
        fetch(`/api/v1/projects/${p.id}/tasks`)
      ]);
      
      if (jobsRes.ok) selectedProject.jobs = await jobsRes.json();
      if (tasksRes.ok) selectedProject.tasks = await tasksRes.json();
    } catch (e) {
      console.error('Failed to fetch project details', e);
    }
  }

  async function triggerJob(jobId: string) {
    try {
      const res = await fetch(`/api/v1/projects/${selectedProject.id}/jobs/${jobId}/trigger`, { method: 'POST' });
      if (res.ok) {
        const data = await res.json();
        activeLogs.set(data.runId, jobId);
        activeLogs = activeLogs; // trigger reactivity
      }
    } catch (e) {
      console.error('Failed to trigger job', e);
    }
  }

  async function triggerTask(taskId: string) {
    try {
      const res = await fetch(`/api/v1/projects/${selectedProject.id}/tasks/${taskId}/trigger`, { method: 'POST' });
      if (res.ok) {
        const data = await res.json();
        activeLogs.set(data.runId, taskId);
        activeLogs = activeLogs;
      }
    } catch (e) {
      console.error('Failed to trigger task', e);
    }
  }

  function closeLogs(runId: string) {
    activeLogs.delete(runId);
    activeLogs = activeLogs;
  }

  onMount(() => {
    fetchProjects();
  });
</script>

<div class="flex h-full w-full">
  <!-- Sidebar -->
  <div class="w-64 border-r border-slate-800 bg-slate-900 flex flex-col">
    <div class="p-4 border-b border-slate-800 flex justify-between items-center">
      <h2 class="font-semibold text-slate-200">Projects</h2>
      <Button variant="ghost" size="icon" onclick={fetchProjects} class="h-8 w-8">
        <RefreshCw class="h-4 w-4" />
      </Button>
    </div>
    <ScrollArea class="flex-1">
      <div class="p-2 space-y-1">
        {#each projects as project}
          <button
            class="w-full text-left px-3 py-2 rounded-md text-sm transition-colors {selectedProject?.id === project.id ? 'bg-indigo-500/10 text-indigo-400 font-medium' : 'text-slate-400 hover:bg-slate-800 hover:text-slate-200'}"
            onclick={() => selectProject(project)}
          >
            {project.name}
          </button>
        {/each}
        {#if projects.length === 0}
          <div class="p-4 text-center text-sm text-slate-500">No projects found</div>
        {/if}
      </div>
    </ScrollArea>
  </div>

  <!-- Main Content -->
  <div class="flex-1 flex flex-col min-w-0 bg-slate-950">
    {#if selectedProject}
      <div class="p-6 border-b border-slate-800 bg-slate-900/50">
        <h2 class="text-2xl font-bold mb-1">{selectedProject.name}</h2>
        {#if selectedProject.desc}
          <p class="text-slate-400 text-sm">{selectedProject.desc}</p>
        {/if}
        <div class="mt-3 flex items-center gap-2 text-xs text-slate-500">
          <code class="px-2 py-1 bg-slate-800 rounded">{selectedProject.file}</code>
        </div>
      </div>

      <ScrollArea class="flex-1 p-6">
        <div class="grid grid-cols-1 lg:grid-cols-2 gap-6 max-w-[1600px] mx-auto pb-10">
          <!-- Jobs -->
          <div class="space-y-4">
            <div class="flex items-center gap-2 mb-2">
              <Server class="w-5 h-5 text-indigo-400" />
              <h3 class="text-xl font-semibold">Jobs</h3>
            </div>
            {#if selectedProject.jobs && selectedProject.jobs.length > 0}
              {#each selectedProject.jobs as job}
                <Card class="bg-slate-900 border-slate-800">
                  <CardHeader class="pb-2">
                    <div class="flex justify-between items-start">
                      <div>
                        <CardTitle class="text-lg flex items-center gap-2">
                          {job.name || job.id}
                          {#if job.cron}
                            <Badge variant="outline" class="border-amber-500/30 text-amber-400 bg-amber-500/10" title="Cron Schedule">
                              <Clock class="w-3 h-3 mr-1" /> {job.cron}
                            </Badge>
                          {/if}
                        </CardTitle>
                        {#if job.desc}
                          <CardDescription>{job.desc}</CardDescription>
                        {/if}
                      </div>
                      <div class="flex gap-2">
                        {#if job.activeRunId}
                          <Button size="sm" variant="outline" class="border-emerald-700 bg-emerald-500/10 text-emerald-400 hover:bg-emerald-500/20" onclick={() => viewLogs(job.activeRunId, `job: ${job.id}`)}>
                            <span class="w-2 h-2 rounded-full bg-emerald-500 animate-pulse mr-2"></span> Running
                          </Button>
                        {/if}
                        <Button size="sm" variant="outline" class="border-slate-700 hover:bg-slate-800" onclick={() => fetchHistory('job', job.id)}>
                          <History class="w-4 h-4 mr-2" /> History
                        </Button>
                        <Button size="sm" onclick={() => triggerJob(job.id)} class="bg-indigo-600 hover:bg-indigo-700">
                          <Play class="w-4 h-4 mr-2" /> Run
                        </Button>
                      </div>
                    </div>
                  </CardHeader>
                  <CardContent>
                    {#if job.steps}
                      <div class="text-sm text-slate-400 mt-2">
                        <span class="font-medium text-slate-300">{job.steps.length}</span> steps
                      </div>
                    {/if}
                  </CardContent>
                </Card>
              {/each}
            {:else}
              <div class="text-slate-500 italic p-4 bg-slate-900/50 rounded-lg border border-slate-800 border-dashed">No jobs defined</div>
            {/if}
          </div>

          <!-- Tasks -->
          <div class="space-y-4">
            <div class="flex items-center gap-2 mb-2">
              <TerminalIcon class="w-5 h-5 text-emerald-400" />
              <h3 class="text-xl font-semibold">Tasks</h3>
            </div>
            {#if selectedProject.tasks && selectedProject.tasks.length > 0}
              {#each selectedProject.tasks as task}
                <Card class="bg-slate-900 border-slate-800">
                  <CardHeader class="pb-2">
                    <div class="flex justify-between items-start">
                      <div>
                        <CardTitle class="text-lg flex items-center gap-2">
                          {task.name || task.id}
                          {#if task.uses === 'ssh'}
                            <Badge variant="outline" class="border-indigo-500/30 text-indigo-400 bg-indigo-500/10">SSH</Badge>
                          {/if}
                        </CardTitle>
                        {#if task.desc}
                          <CardDescription>{task.desc}</CardDescription>
                        {/if}
                      </div>
                      <div class="flex gap-2">
                        {#if task.uses === 'ssh' && task.hosts && task.hosts.length > 0}
                          <!-- For simplicity we just take the first host to connect interactive ssh -->
                          <Button size="sm" variant="outline" class="border-slate-700 hover:bg-slate-800" onclick={() => dispatch('ssh', { host: task.hosts[0], projectId: selectedProject.id })}>
                            <TerminalIcon class="w-4 h-4 mr-2" /> Connect
                          </Button>
                        {/if}
                        {#if task.activeRunId}
                          <Button size="sm" variant="outline" class="border-emerald-700 bg-emerald-500/10 text-emerald-400 hover:bg-emerald-500/20" onclick={() => viewLogs(task.activeRunId, `task: ${task.id}`)}>
                            <span class="w-2 h-2 rounded-full bg-emerald-500 animate-pulse mr-2"></span> Running
                          </Button>
                        {/if}
                        <Button size="sm" variant="outline" class="border-slate-700 hover:bg-slate-800" onclick={() => fetchHistory('task', task.id)}>
                          <History class="w-4 h-4 mr-2" /> History
                        </Button>
                        <Button size="sm" onclick={() => triggerTask(task.id)} class="bg-emerald-600 hover:bg-emerald-700">
                          <Play class="w-4 h-4 mr-2" /> Run
                        </Button>
                      </div>
                    </div>
                  </CardHeader>
                </Card>
              {/each}
            {:else}
              <div class="text-slate-500 italic p-4 bg-slate-900/50 rounded-lg border border-slate-800 border-dashed">No tasks defined</div>
            {/if}
          </div>
        </div>
      </ScrollArea>
    {:else}
      <div class="flex-1 flex items-center justify-center text-slate-500">
        <div class="text-center">
          <Layers class="w-16 h-16 mx-auto mb-4 opacity-20" />
          <p>Select a project to view its jobs and tasks</p>
        </div>
      </div>
    {/if}
  </div>

  <!-- Logs Overlay -->
  {#if activeLogs.size > 0}
    <div class="fixed bottom-0 right-0 w-full md:w-[600px] lg:w-[800px] max-h-[50vh] flex flex-col pointer-events-none p-4 z-40">
      <div class="pointer-events-auto flex flex-col gap-4">
        {#each Array.from(activeLogs.entries()) as [runId, name]}
          <div class="bg-slate-900 border border-slate-700 rounded-lg shadow-2xl overflow-hidden flex flex-col h-64">
            <div class="bg-slate-800 px-3 py-2 flex justify-between items-center border-b border-slate-700">
              <div class="font-mono text-xs flex items-center gap-2">
                <span class="w-2 h-2 rounded-full bg-emerald-500 animate-pulse"></span>
                Logs: <span class="text-indigo-300">{name}</span>
              </div>
              <div class="flex items-center gap-3">
                <a 
                  href={`/api/v1/runs/${runId}/logs`} 
                  download={`run-${runId}.log`}
                  class="text-xs text-slate-400 hover:text-white transition-colors"
                  title="Download Logs"
                >
                  Download
                </a>
                <button class="text-slate-400 hover:text-white transition-colors" onclick={() => closeLogs(runId)}>
                  <XCircle class="w-4 h-4" />
                </button>
              </div>
            </div>
            <div class="flex-1 overflow-hidden relative">
              <JobLogs {runId} />
            </div>
          </div>
        {/each}
      </div>
    </div>
  {/if}

  <!-- History Dialog -->
  <Dialog.Root bind:open={historyOpen}>
    <Dialog.Content class="sm:max-w-[700px] bg-slate-950 border-slate-800 text-slate-200">
      <Dialog.Header>
        <Dialog.Title>Run History - {historyTitle}</Dialog.Title>
        <Dialog.Description>
          Recent executions for this {historyTitle.split(':')[0].toLowerCase()}.
        </Dialog.Description>
      </Dialog.Header>
      
      <ScrollArea class="h-[400px] mt-4 border rounded-md border-slate-800">
        <Table.Root>
          <Table.Header class="bg-slate-900 sticky top-0">
            <Table.Row class="border-slate-800 hover:bg-slate-900">
              <Table.Head class="w-[100px]">Status</Table.Head>
              <Table.Head>Started</Table.Head>
              <Table.Head>Duration</Table.Head>
              <Table.Head>Trigger</Table.Head>
              <Table.Head class="text-right">Action</Table.Head>
            </Table.Row>
          </Table.Header>
          <Table.Body>
            {#if historyRuns.length === 0}
              <Table.Row class="border-slate-800">
                <Table.Cell colspan={5} class="text-center text-slate-500 py-8">No runs found</Table.Cell>
              </Table.Row>
            {:else}
              {#each historyRuns as run}
                <Table.Row class="border-slate-800">
                  <Table.Cell>
                    {#if run.Status === 'success'}
                      <Badge variant="outline" class="bg-emerald-500/10 text-emerald-400 border-emerald-500/20">Success</Badge>
                    {:else if run.Status === 'failed'}
                      <Badge variant="outline" class="bg-red-500/10 text-red-400 border-red-500/20">Failed</Badge>
                    {:else}
                      <Badge variant="outline" class="bg-indigo-500/10 text-indigo-400 border-indigo-500/20 capitalize">{run.Status}</Badge>
                    {/if}
                  </Table.Cell>
                  <Table.Cell class="text-slate-400">
                    {new Date(run.CreatedAt).toLocaleString()}
                  </Table.Cell>
                  <Table.Cell class="text-slate-400">
                    {formatDuration(run.CreatedAt, run.CompletedAt)}
                  </Table.Cell>
                  <Table.Cell class="text-slate-400 text-sm">
                    {#if run.TriggeredBy}
                      {run.TriggeredBy}
                    {:else}
                      -
                    {/if}
                  </Table.Cell>
                  <Table.Cell class="text-right">
                    <Button variant="ghost" size="sm" onclick={() => viewLogs(run.ID, historyTitle.split(': ')[1])}>
                      View Logs
                    </Button>
                  </Table.Cell>
                </Table.Row>
              {/each}
            {/if}
          </Table.Body>
        </Table.Root>
      </ScrollArea>
    </Dialog.Content>
  </Dialog.Root>
</div>
