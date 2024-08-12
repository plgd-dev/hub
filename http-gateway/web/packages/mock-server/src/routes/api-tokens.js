const { checkError, loadResponseStreamFromFile, loadResponseFromFile } = require('../utils')
const escapeHtml = require('escape-html')
const express = require('express')
const path = require('path')
const get = require('lodash/get')
const axios = require('axios')

const router = express.Router()

let tokenAdded = false
let tokenDeleted = false

router.get('/api/v1/api-reset', (req, res) => {
    try {
        checkError(req, res)

        tokenAdded = false
        tokenDeleted = false

        res.send('OK')
    } catch (e) {
        res.status(500).send(escapeHtml(e.toString()))
    }
})

router.get('/.well-known/openid-configuration', (req, res) => {
    try {
        checkError(req, res)
        axios.get('https://try.plgd.cloud/m2m-oauth-server/.well-known/openid-configuration').then((r) => res.send(r.data))
    } catch (e) {
        res.status(500).send(e.toString())
    }
})

router.post('/api/v1/tokens', (req, res) => {
    try {
        checkError(req, res)
        loadResponseFromFile(path.join('api-tokens', 'create-no-expiration.json'), res)
    } catch (e) {
        res.status(500).send(e.toString())
    }
})

router.get('/api/v1/tokens', (req, res) => {
    try {
        checkError(req, res)
        const filter = get(req.query, 'idFilter')

        // detail page
        if (filter) {
            loadResponseFromFile(path.join('api-tokens', 'detail', `${filter}.json`), res)
        } else if (tokenAdded) {
            // list page after add
            loadResponseStreamFromFile(path.join('api-tokens', 'list', `listAdd.json`), res)
        } else if (tokenDeleted) {
            // list page after delete
            loadResponseStreamFromFile(path.join('api-tokens', 'list', `listEmpty.json`), res)
        } else {
            // list page default
            loadResponseStreamFromFile(path.join('api-tokens', 'list', `list.json`), res)
        }
    } catch (e) {
        res.status(500).send(escapeHtml(e.toString()))
    }
})

router.delete('/api/v1/tokens', (req, res) => {
    try {
        checkError(req, res)
        tokenDeleted = true
        res.status(200).send('OK')
    } catch (e) {
        res.status(500).send(escapeHtml(e.toString()))
    }
})

module.exports = router
