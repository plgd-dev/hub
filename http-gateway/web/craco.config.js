const path = require('path')
/* eslint-disable */
const { CracoAliasPlugin } = require('react-app-alias-ex')

module.exports = {
  webpack: {
    alias: {
      '@': path.resolve(__dirname, 'src/'),
      '@shared-ui': path.resolve(__dirname, '../shared-ui/src/'),
    },
  },
  plugins: [
    {
      plugin: CracoAliasPlugin,
      options: {},
    },
  ],
}
