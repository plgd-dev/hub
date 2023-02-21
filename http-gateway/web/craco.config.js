const path = require('path')
const webpack = require('webpack');
/* eslint-disable */
const { CracoAliasPlugin } = require('react-app-alias-ex')

module.exports = {
  webpack: {
    alias: {
      '@': path.resolve(__dirname, 'src/'),
      '@shared-ui': path.resolve(__dirname, '../shared-ui/src/'),
    },
    plugins: {
      add: [
        new webpack.DefinePlugin({
          PRODUCTION: JSON.stringify(true),
          'process.env.NODE_ENV': JSON.stringify(process.env.NODE_ENV),
        })
      ]
    }
  },
  plugins: [
    {
      plugin: CracoAliasPlugin,
      options: {},
    },
  ],
}
