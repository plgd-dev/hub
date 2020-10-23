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
              'architecture/1-authorization-server',
              'architecture/2-resource-aggregate',
              'architecture/3-resource-directory',
              'architecture/4-coap-gateway',
            ]
          }
        ]
      }
    },
    dest: "dist",
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