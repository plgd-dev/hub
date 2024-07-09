const { checkError, loadResponseStreamFromFile, loadResponseFromFile } = require('../utils')
const get = require('lodash/get')
const path = require('path')
const escapeHtml = require('escape-html')
const express = require('express')

const router = express.Router()

router.get('/api/v1/hubs', (req, res) => {
    try {
        checkError(req, res)
        const httpIdFilter = get(req.query, 'idFilter', null)

        // detail configuration page
        if (httpIdFilter) {
            loadResponseFromFile(path.join('dps', 'linked-hubs', 'detail', `${httpIdFilter}.json`), res)
        } else {
            loadResponseStreamFromFile(path.join('dps', 'linked-hubs', 'list', `list.json`), res)
        }
    } catch (e) {
        res.status(500).send(escapeHtml(e.toString()))
    }
})

router.get('/api/v1/enrollment-groups', (req, res) => {
    try {
        checkError(req, res)
        const httpIdFilter = get(req.query, 'idFilter', null)

        // detail configuration page
        if (httpIdFilter) {
            loadResponseFromFile(path.join('dps', 'enrollment-groups', 'detail', `${httpIdFilter}.json`), res)
        } else {
            loadResponseStreamFromFile(path.join('dps', 'enrollment-groups', 'list', `list.json`), res)
        }
    } catch (e) {
        res.status(500).send(escapeHtml(e.toString()))
    }
})

router.get('/api/v1/provisioning-records', (req, res) => {
    try {
        checkError(req, res)
        const httpIdFilter = get(req.query, 'idFilter', null)

        // detail configuration page
        if (httpIdFilter) {
            loadResponseFromFile(path.join('dps', 'provisioning-records', 'detail', `${httpIdFilter}.json`), res)
        } else {
            loadResponseStreamFromFile(path.join('dps', 'provisioning-records', 'list', `list.json`), res)
        }
    } catch (e) {
        res.status(500).send(escapeHtml(e.toString()))
    }
})

module.exports = router
