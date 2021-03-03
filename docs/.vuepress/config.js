module.exports = {
    title: 'plgd',
    description: 'Secure and Interoperable Internet of Things',
    themeConfig: {
      logo: '/img/logo-long.svg',
      repo: 'plgd-dev/cloud',
      docsRepo: 'plgd-dev/cloud',
      editLinks: true,
      editLinkText: 'Help us improve this page!',
      nav: [
        { text: 'Guide', link: '/guide/' },
        { text: 'Chat with us', link: 'https://gitter.im/ocfcloud/Lobby' },
        { text: 'Changelog', link: 'https://github.com/plgd-dev/cloud/releases' }
      ],
      sidebarDepth: 1,
      sidebar: {
        '/guide/': [
          {
            title: 'Getting Started',
            children: [
              'getting-started/1-deploy',
              'getting-started/2-onboard',
              'getting-started/3-interact'
            ]
          },
          {
            title: 'Architecture',
            children: [
              'architecture/domain-overview',
              'architecture/system-overview'
            ]
          },
          {
            title: 'Deployment',
            children: [
              'deployment/authorization-server',
              'deployment/resource-aggregate',
              'deployment/resource-directory',
              'deployment/coap-gateway',
              'deployment/cloud2cloud-connector',
              'deployment/cloud2cloud-gateway',
            ]
          },
          {
            title: 'Developing with plgd',
            sidebarDepth: 1,
            children: [
              'developing/resources',
              'developing/dashboard'
            ]
          }
        ]
      }
    },
    dest: "dist",
    extendMarkdown: md => {
      md.set({ breaks: true })
      md.use(require('markdown-it-plantuml'))
      md.use(require('markdown-it-imsize'))
    },
    plugins: [
      '@vuepress/medium-zoom',
      [
        '@vuepress/google-analytics',
        {
          'ga': 'UA-165501387-1'
        }
      ]  
    ] 
  }