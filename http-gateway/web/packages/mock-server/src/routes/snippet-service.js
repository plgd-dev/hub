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
let conditionDeleted = false
let conditionUpdated = false
let appliedConfigurationsDeleted = false

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

router.get('/api/v1/applied-configurations/api-reset', (req, res) => {
    try {
        checkError(req, res)

        appliedConfigurationsDeleted = false

        res.send('OK')
    } catch (e) {
        res.status(500).send(escapeHtml(e.toString()))
    }
})

router.get('/api/v1/conditions/api-reset', (req, res) => {
    try {
        checkError(req, res)
        conditionDeleted = false
        conditionUpdated = false

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

        if (appliedConfigurationsDeleted) {
            loadResponseStreamFromFile(path.join('snippet-service', 'applied-configurations', 'list', 'listEmpty.json'), res)
        } else if (httpConfigurationIdFilter) {
            // detail configuration page
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

const removeVersionFromFilter = (filters) => {
    if (!filters) {
        return [filters, null]
    }

    const filter = filters?.split('/')
    let version = null

    if (filter && !filter[filter.length - 1].includes('-')) {
        version = filter.pop()
    }

    return [filter.join('/'), version]
}

const parseFilters = (query, key) => {
    let filters = get(query, key, null)

    if (Array.isArray(filters)) {
        filters = uniq(filters)[0]
    }

    return removeVersionFromFilter(filters?.replace('/all', ''))
}

router.get('/api/v1/configurations', (req, res) => {
    try {
        checkError(req, res)

        const [filter, version] = parseFilters(req.query, 'httpIdFilter')

        // detail page
        if (filter) {
            loadResponseStreamFromFile(path.join('snippet-service', 'configurations', 'detail', `${filter}.json`), res, version)
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

router.delete('/api/v1/configurations/applied', (req, res) => {
    try {
        checkError(req, res)
        appliedConfigurationsDeleted = true
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
        const [filter, version] = parseFilters(req.query, 'httpIdFilter')
        const [filterD, versionD] = parseFilters(req.query, 'configurationIdFilter')

        // detail page
        if (filter) {
            loadResponseStreamFromFile(path.join('snippet-service', 'conditions', 'detail', `${filter}.json`), res, version)
        } else if (filterD) {
            // list page after delete
            loadResponseStreamFromFile(path.join('snippet-service', 'conditions', 'detail', `${filterD}.json`), res, versionD)
        } else if (conditionDeleted) {
            // list page after delete
            loadResponseStreamFromFile(path.join('snippet-service', 'conditions', 'list', `listEmpty.json`), res)
        } else if (conditionUpdated) {
            // list page after delete
            loadResponseStreamFromFile(path.join('snippet-service', 'conditions', 'list', `listUpdate.json`), res)
        } else {
            // list page
            loadResponseStreamFromFile(path.join('snippet-service', 'conditions', 'list', `list.json`), res)
        }
    } catch (e) {
        res.status(500).send(escapeHtml(e.toString()))
    }
})

router.post('/api/v1/conditions', (req, res) => {
    try {
        checkError(req, res)
        res.status(200).send('OK')
    } catch (e) {
        res.status(500).send(escapeHtml(e.toString()))
    }
})

router.delete('/api/v1/conditions', (req, res) => {
    try {
        checkError(req, res)
        conditionDeleted = true
        res.status(200).send('OK')
    } catch (e) {
        res.status(500).send(escapeHtml(e.toString()))
    }
})

router.put('/api/v1/conditions/:conditionId', configurationIdCheck, (req, res) => {
    try {
        checkError(req, res)
        conditionUpdated = true
        res.status(200).send('OK')
    } catch (e) {
        res.status(500).send(escapeHtml(e.toString()))
    }
})

module.exports = router
