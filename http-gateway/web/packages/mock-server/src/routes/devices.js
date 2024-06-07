const express = require('express')
const { checkError, loadResponseFromFile } = require('../utils')
const path = require('path')
const { check } = require('express-validator')

const router = express.Router()

let deletedDevice = false
let resourceColorUpdatedValue = false

const deviceIdCheck = [check('deviceId').notEmpty().withMessage('Device ID must be alphanumeric')]

router.get('/api/v1/devices/api-reset', (req, res) => {
    try {
        checkError(req, res)

        deletedDevice = false
        resourceColorUpdatedValue = false

        res.send('OK')
    } catch (e) {
        res.status(500).send(e.toString())
    }
})

router.get('/api/v1/devices', (req, res) => {
    try {
        checkError(req, res)

        if (deletedDevice) {
            loadResponseFromFile(path.join('devices', 'list', 'list-deleted-state.json'), res)
        } else {
            loadResponseFromFile(path.join('devices', 'list', 'list.json'), res)
        }
    } catch (e) {
        res.status(500).send(e.toString())
    }
})

router.delete('/api/v1/devices', (req, res) => {
    try {
        checkError(req, res)
        deletedDevice = true

        res.send('OK')
    } catch (e) {
        res.status(500).send(e.toString())
    }
})

router.get('/api/v1/devices/:deviceId', deviceIdCheck, (req, res) => {
    try {
        checkError(req, res)
        loadResponseFromFile(path.join('devices', 'detail', `${req.params['deviceId']}.json`), res)
    } catch (e) {
        res.status(500).send(e.toString())
    }
})

router.get('/api/v1/devices/:deviceId/pending-commands', deviceIdCheck, (req, res) => {
    try {
        checkError(req, res)
        res.send()
    } catch (e) {
        res.status(500).send(e.toString())
    }
})

router.get('/api/v1/devices/:deviceId/resources', deviceIdCheck, (req, res) => {
    try {
        checkError(req, res)
        res.send()
    } catch (e) {
        res.status(500).send(e.toString())
    }
})

router.put('/api/v1/devices/:deviceId/metadata', deviceIdCheck, (req, res) => {
    try {
        checkError(req, res)
        res.send()
    } catch (e) {
        res.status(500).send(e.toString())
    }
})

// change device name
router.put('/api/v1/devices/:deviceId/resources/oc/con', deviceIdCheck, (req, res) => {
    try {
        checkError(req, res)
        res.send({ n: 'New Device Name' })
    } catch (e) {
        res.status(500).send(e.toString())
    }
})

// resource detail
router.get('/api/v1/devices/:deviceId/resources/light/1', deviceIdCheck, (req, res) => {
    try {
        checkError(req, res)
        loadResponseFromFile(path.join('devices', 'detail', `${req.params['deviceId']}-resources-light-1.json`), res)
    } catch (e) {
        res.status(500).send(e.toString())
    }
})

// resource detail update
router.put('/api/v1/devices/:deviceId/resources/light/1', deviceIdCheck, (req, res) => {
    try {
        checkError(req, res)
        loadResponseFromFile(path.join('devices', 'detail', `${req.params['deviceId']}-resources-light-1.json`), res)
    } catch (e) {
        res.status(500).send(e.toString())
    }
})

// resource detail wot
router.get('/api/v1/devices/:deviceId/resources/.well-known/wot', deviceIdCheck, (req, res) => {
    try {
        checkError(req, res)
        loadResponseFromFile(path.join('devices', 'detail', `${req.params['deviceId']}-resources-well-known-wot.json`), res)
    } catch (e) {
        res.status(500).send(e.toString())
    }
})

// resource detail color
router.get('/api/v1/devices/:deviceId/resources/color', deviceIdCheck, (req, res) => {
    try {
        checkError(req, res)

        if (resourceColorUpdatedValue) {
            loadResponseFromFile(path.join('devices', 'detail', `${req.params['deviceId']}-resources-color-update.json`), res)
        } else {
            loadResponseFromFile(path.join('devices', 'detail', `${req.params['deviceId']}-resources-color.json`), res)
        }
    } catch (e) {
        res.status(500).send(e.toString())
    }
})

// resource detail color update
router.put('/api/v1/devices/:deviceId/resources/color', deviceIdCheck, (req, res) => {
    try {
        checkError(req, res)
        resourceColorUpdatedValue = true
        loadResponseFromFile(path.join('devices', 'detail', `${req.params['deviceId']}-resources-color-update.json`), res)
    } catch (e) {
        res.status(500).send(e.toString())
    }
})

router.get('/api/v1/resource-links', deviceIdCheck, (req, res) => {
    try {
        checkError(req, res)
        loadResponseFromFile(path.join('devices', 'detail', `${req.query['device_id_filter']}-resource-links.json`), res)
    } catch (e) {
        res.status(500).send(e.toString())
    }
})

router.get('/api/v1/provisioning-records', deviceIdCheck, (req, res) => {
    try {
        checkError(req, res)
        loadResponseFromFile(path.join('devices', 'detail', `${req.query['deviceIdFilter']}-provisioning-records.json`), res)
    } catch (e) {
        res.status(500).send(e.toString())
    }
})

router.get('/api/v1/signing/records', deviceIdCheck, (req, res) => {
    try {
        checkError(req, res)
        loadResponseFromFile(path.join('devices', 'detail', `${req.query['deviceIdFilter']}-signin-records.json`), res)
    } catch (e) {
        res.status(500).send(e.toString())
    }
})

module.exports = router
