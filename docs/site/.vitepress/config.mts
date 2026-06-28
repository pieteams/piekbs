import { defineConfig } from 'vitepress'

export default defineConfig({
  title: 'WikiLoop',
  description: 'A knowledge search engine for agents',
  base: '/wikiloop/',
  sitemap: {
    hostname: 'https://jasen215.github.io/wikiloop/',
  },
  cleanUrls: true,
  lastUpdated: true,
  head: [
    ['script', { async: '', src: 'https://www.googletagmanager.com/gtag/js?id=G-FD9FS6Q7GQ' }],
    ['script', {}, `
      window.dataLayer = window.dataLayer || [];
      function gtag(){dataLayer.push(arguments);}
      gtag('js', new Date());
      gtag('config', 'G-FD9FS6Q7GQ');
    `]
  ],
  themeConfig: {
    socialLinks: [
      { icon: 'github', link: 'https://github.com/jasen215/wikiloop' }
    ],
    footer: {
      message: 'Released under the MIT License.',
      copyright: 'Copyright © 2024-present Jasen Han'
    },
  },
  locales: {
    root: {
      label: 'English',
      lang: 'en',
      themeConfig: {
        nav: [
          {
            text: 'Getting Started',
            link: '/getting-started/what-is-wikiloop',
            activeMatch: '/getting-started/',
          },
          {
            text: 'Guide',
            link: '/guide/how-agents-use',
            activeMatch: '/guide/',
          },
          {
            text: 'Reference',
            link: '/reference/cli',
            activeMatch: '/reference/',
          },
          {
            text: 'Ecosystem',
            link: '/ecosystem/rag-systems',
            activeMatch: '/ecosystem/',
          },
        ],
        sidebar: {
          '/getting-started/': [
            {
              text: 'Getting Started',
              items: [
                { text: 'What is WikiLoop', link: '/getting-started/what-is-wikiloop' },
                { text: 'Installation', link: '/getting-started/installation' },
                { text: 'Quick Start', link: '/getting-started/quick-start' },
              ]
            }
          ],
          '/guide/': [
            {
              text: 'Guide',
              items: [
                { text: 'How Agents Use WikiLoop', link: '/guide/how-agents-use' },
                { text: 'Knowledge Pipeline', link: '/guide/knowledge-pipeline' },
                { text: 'MCP Server', link: '/guide/mcp-server' },
                { text: 'Schema & Templates', link: '/guide/schema-templates' },
              ]
            }
          ],
          '/reference/': [
            {
              text: 'Reference',
              items: [
                { text: 'CLI', link: '/reference/cli' },
                { text: 'MCP Tools', link: '/reference/mcp-tools' },
                { text: 'Config', link: '/reference/config' },
              ]
            }
          ],
          '/ecosystem/': [
            {
              text: 'Systems',
              items: [
                { text: 'RAG Systems', link: '/ecosystem/rag-systems' },
                { text: 'LLM Wiki Systems', link: '/ecosystem/llm-wiki-systems' },
              ]
            },
            {
              text: 'Technologies',
              items: [
                { text: 'RAG Technologies', link: '/ecosystem/rag-technologies' },
                { text: 'LLM Wiki Technologies', link: '/ecosystem/llm-wiki-technologies' },
                { text: 'Paradigm Comparison', link: '/ecosystem/paradigm-comparison' },
              ]
            }
          ],
        }
      }
    },
    'zh-CN': {
      label: '简体中文',
      lang: 'zh-CN',
      link: '/zh-CN/',
      themeConfig: {
        nav: [
          {
            text: '快速开始',
            link: '/zh-CN/getting-started/what-is-wikiloop',
            activeMatch: '/zh-CN/getting-started/',
          },
          {
            text: '指南',
            link: '/zh-CN/guide/how-agents-use',
            activeMatch: '/zh-CN/guide/',
          },
          {
            text: '参考',
            link: '/zh-CN/reference/cli',
            activeMatch: '/zh-CN/reference/',
          },
          {
            text: '生态',
            link: '/zh-CN/ecosystem/rag-systems',
            activeMatch: '/zh-CN/ecosystem/',
          },
        ],
        sidebar: {
          '/zh-CN/getting-started/': [
            {
              text: '快速开始',
              items: [
                { text: '什么是 WikiLoop', link: '/zh-CN/getting-started/what-is-wikiloop' },
                { text: '安装', link: '/zh-CN/getting-started/installation' },
                { text: '快速入门', link: '/zh-CN/getting-started/quick-start' },
              ]
            }
          ],
          '/zh-CN/guide/': [
            {
              text: '指南',
              items: [
                { text: 'Agent 如何使用 WikiLoop', link: '/zh-CN/guide/how-agents-use' },
                { text: '知识管道', link: '/zh-CN/guide/knowledge-pipeline' },
                { text: 'MCP 服务器', link: '/zh-CN/guide/mcp-server' },
                { text: 'Schema 与模板', link: '/zh-CN/guide/schema-templates' },
              ]
            }
          ],
          '/zh-CN/reference/': [
            {
              text: '参考',
              items: [
                { text: 'CLI', link: '/zh-CN/reference/cli' },
                { text: 'MCP 工具', link: '/zh-CN/reference/mcp-tools' },
                { text: '配置', link: '/zh-CN/reference/config' },
              ]
            }
          ],
          '/zh-CN/ecosystem/': [
            {
              text: '系统列表',
              items: [
                { text: 'RAG 系统', link: '/zh-CN/ecosystem/rag-systems' },
                { text: 'LLM Wiki 系统', link: '/zh-CN/ecosystem/llm-wiki-systems' },
              ]
            },
            {
              text: '关键技术',
              items: [
                { text: 'RAG 关键技术', link: '/zh-CN/ecosystem/rag-technologies' },
                { text: 'LLM Wiki 关键技术', link: '/zh-CN/ecosystem/llm-wiki-technologies' },
                { text: '范式对比', link: '/zh-CN/ecosystem/paradigm-comparison' },
              ]
            }
          ],
        }
      }
    }
  }
})
