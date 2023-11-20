const path = require('path')
const webpack = require('webpack');
/* eslint-disable */
const { CracoAliasPlugin } = require('react-app-alias-ex')

module.exports = {
  webpack: {
    alias: {
      '@': path.resolve(__dirname, 'src/'),
      '@shared-ui': path.resolve(__dirname, '../packages/shared-ui/src/'),
      '@shared-ui/*': path.resolve(__dirname, '../packages/shared-ui/src/*'),
    },
    plugins: {
      add: [
        new webpack.DefinePlugin({
          PRODUCTION: JSON.stringify(true),
          process: {
          //   env: {
          //     // NODE_ENV: 'production'
          //   }
          },
          'process.env.NODE_ENV': JSON.stringify(process.env.NODE_ENV),
        })
      ]
    }
  },
  babel: {
    plugins: [
      ["@emotion/babel-plugin"],
      ["@babel/plugin-transform-react-jsx", {
        "runtime": "automatic",
        "importSource": "@emotion/react"
      }]
    ],
  },
  plugins: [
    {
      plugin: CracoAliasPlugin,
      options: {},
    },
  ],
}
