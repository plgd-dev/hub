'use strict'

module.exports = function(api) {
  api.cache(true)
  api.assertVersion("^7.19.3")

  const presets = [
    [
      "react-app",
      {
        "flow": false,
        "typescript": true,
        "helpers": false
      }
    ]
  ]

  const plugins = [
    ["@emotion/babel-plugin"],
    ["@babel/plugin-transform-react-jsx", {
      "runtime": "automatic",
      "importSource": "@emotion/react"
    }]
  ]

  return {
    presets,
    plugins
  }
}
