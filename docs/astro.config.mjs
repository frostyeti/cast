// @ts-check
import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';

// https://astro.build/config
export default defineConfig({
	site: 'https://cast.frostyeti.com',
	integrations: [
		starlight({
			title: 'Cast Docs',
			social: [
				{
					icon: 'github',
					label: 'GitHub',
					href: 'https://github.com/frostyeti/cast',
				},
			],
			sidebar: [
				{
					label: 'Guides',
					items: [
						{ label: 'Getting Started', slug: 'guides/getting-started' },
						{ label: 'Examples', slug: 'guides/examples' },
					],
				},
				{
					label: 'Reference',
					items: [
						{ label: 'CLI Reference', slug: 'reference/cli' },
					],
				},
				{
					label: 'Issues',
					autogenerate: { directory: 'issues' },
				},
			],
		}),
	],
});
