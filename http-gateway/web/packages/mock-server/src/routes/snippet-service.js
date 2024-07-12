const express = require('express')
const { loadResponseStreamFromFile, checkError } = require('../utils')
const path = require('path')
const escapeHtml = require('escape-html')
const get = require('lodash/get')

const router = express.Router()

let configurationsAdd = false

router.get('/api/v1/configurations/applied', (req, res) => {
    try {
        checkError(req, res)
        const httpConfigurationIdFilter = get(req.query, 'httpConfigurationIdFilter', null)?.replace('/all', '')

        // detail configuration page
        if (httpConfigurationIdFilter) {
            loadResponseStreamFromFile(
                path.join('snippet-service', 'applied-configurations', 'list', `httpConfigurationIdFilter-${httpConfigurationIdFilter}.json`),
                res
            )
        } else {
            // res.send([])
        }
    } catch (e) {
        res.status(500).send(escapeHtml(e.toString()))
    }
})

router.get('/api/v1/configurations', (req, res) => {
    try {
        checkError(req, res)
        const filter = get(req.query, 'httpIdFilter', null)?.replace('/all', '')

        // detail page
        if (filter) {
            loadResponseStreamFromFile(path.join('snippet-service', 'configurations', 'detail', `${filter}.json`), res)
        }
        // list page after add
        if (configurationsAdd) {
            loadResponseStreamFromFile(path.join('snippet-service', 'configurations', 'list', `listAdd.json`), res)
        } else {
            // list page default
            loadResponseStreamFromFile(path.join('snippet-service', 'configurations', 'list', `list.json`), res)
        }
    } catch (e) {
        res.status(500).send(escapeHtml(e.toString()))
    }
})

router.post('/api/v1/configurations', (req, res) => {
    try {
        checkError(req, res)

        configurationsAdd = this

        res.status(200).send({ id: '1a53e16f-b533-4c26-9150-e2c30065ab27' })
    } catch (e) {
        res.status(500).send(escapeHtml(e.toString()))
    }
})

router.get('/api/v1/conditions', (req, res) => {
    try {
        checkError(req, res)
        const filter = get(req.query, 'httpIdFilter', null)?.replace('/all', '')

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
