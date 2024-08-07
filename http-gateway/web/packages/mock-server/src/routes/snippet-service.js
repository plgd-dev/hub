const express = require('express')
const { loadResponseStreamFromFile, checkError } = require('../utils')
const path = require('path')
const escapeHtml = require('escape-html')
const get = require('lodash/get')
const { check } = require('express-validator')
const { uniq } = require('lodash')

const router = express.Router()

let configurationsAdd = false
let configurationsDeleted = false

const configurationIdCheck = [check('configurationId').notEmpty().withMessage('Configuration ID must be alphanumeric')]

router.get('/api/v1/configurations/api-reset', (req, res) => {
    try {
        checkError(req, res)

        configurationsAdd = false
        configurationsDeleted = false

        res.send('OK')
    } catch (e) {
        res.status(500).send(escapeHtml(e.toString()))
    }
})

router.get('/api/v1/configurations/applied', (req, res) => {
    try {
        checkError(req, res)
        const extractFilter = (query, key) => get(query, key, null)?.replace('/all', '')
        const httpConfigurationIdFilter = extractFilter(req.query, 'httpConfigurationIdFilter')
        const idFilter = extractFilter(req.query, 'idFilter')

        // detail configuration page
        if (httpConfigurationIdFilter) {
            loadResponseStreamFromFile(
                path.join('snippet-service', 'applied-configurations', 'list', `httpConfigurationIdFilter-${httpConfigurationIdFilter}.json`),
                res
            )
        } else if (idFilter) {
            loadResponseStreamFromFile(path.join('snippet-service', 'applied-configurations', 'detail', `idFilter-${idFilter}.json`), res)
        } else {
            loadResponseStreamFromFile(path.join('snippet-service', 'applied-configurations', 'list', 'list.json'), res)
        }
    } catch (e) {
        res.status(500).send(escapeHtml(e.toString()))
    }
})

const parseFilters = (query, key) => {
    const filters = get(query, key, null)

    if (Array.isArray(filters)) {
        return uniq(filters)
    } else {
        return filters?.replace('/all', '')?.replace(/\/[0-9]+/g, '')
    }
}

router.get('/api/v1/configurations', (req, res) => {
    try {
        checkError(req, res)

        const parsedFilter = parseFilters(req.query, 'httpIdFilter')
        const filter = Array.isArray(parsedFilter) ? parsedFilter[0] : parsedFilter

        // detail page
        if (filter) {
            loadResponseStreamFromFile(path.join('snippet-service', 'configurations', 'detail', `${filter}.json`), res)
        } else if (configurationsAdd) {
            // list page after add
            loadResponseStreamFromFile(path.join('snippet-service', 'configurations', 'list', `listAdd.json`), res)
        } else if (configurationsDeleted) {
            // list page after delete
            loadResponseStreamFromFile(path.join('snippet-service', 'configurations', 'list', `listEmpty.json`), res)
        } else {
            // list page default
            loadResponseStreamFromFile(path.join('snippet-service', 'configurations', 'list', `list.json`), res)
        }
    } catch (e) {
        res.status(500).send(escapeHtml(e.toString()))
    }
})

router.delete('/api/v1/configurations', (req, res) => {
    try {
        checkError(req, res)
        configurationsDeleted = true
        res.status(200).send('OK')
    } catch (e) {
        res.status(500).send(escapeHtml(e.toString()))
    }
})

router.post('/api/v1/configurations', (req, res) => {
    try {
        checkError(req, res)

        configurationsAdd = true

        res.status(200).send({ id: '1a53e16f-b533-4c26-9150-e2c30065ab27' })
    } catch (e) {
        res.status(500).send(escapeHtml(e.toString()))
    }
})

router.post('/api/v1/configurations/:configurationId', configurationIdCheck, (req, res) => {
    try {
        checkError(req, res)
        res.status(200).send('OK')
    } catch (e) {
        res.status(500).send(escapeHtml(e.toString()))
    }
})

router.put('/api/v1/configurations/:configurationId', configurationIdCheck, (req, res) => {
    try {
        checkError(req, res)
        res.status(200).send('OK')
    } catch (e) {
        res.status(500).send(escapeHtml(e.toString()))
    }
})

router.get('/api/v1/conditions', (req, res) => {
    try {
        checkError(req, res)
        const filter = get(req.query, 'httpIdFilter', null)
            ?.replace('/all', '')
            ?.replace(/\/[0-9]+/g, '')

        // detail page
        if (filter) {
            loadResponseStreamFromFile(path.join('snippet-service', 'conditions', 'detail', `${filter}.json`), res)
        } else {
            // list page
            loadResponseStreamFromFile(path.join('snippet-service', 'conditions', 'list', `list.json`), res)
        }
    } catch (e) {
        res.status(500).send(escapeHtml(e.toString()))
    }
})

module.exports = router
