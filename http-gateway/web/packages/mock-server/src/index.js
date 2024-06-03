const express = require('express')
const { check, validationResult } = require('express-validator')
const cors = require('cors')
const path = require('path')

const app = express()
const port = 8181

let deletedDevice = false
let resourceColorUpdatedValue = false

const deviceIdCheck = [check('deviceId').notEmpty().withMessage('Device ID must be alphanumeric')]

const checkError = (req, res) => {
    const errors = validationResult(req)
    if (!errors.isEmpty()) {
        return res.status(400).json({ errors: errors.array() })
    }

    console.log(`${req.method}`, req.url)
}

app.use(
    cors({
        origin: '*',
        methods: ['GET', 'POST', 'DELETE', 'UPDATE', 'PUT', 'PATCH'],
    })
)

const loadResponseFromFile = (file, res) => {
    const targetDirectory = `${__dirname}/data`

    res.sendFile(file, { root: targetDirectory })
}

// ----- DEVICES -----
app.get('/api/v1/devices', function (req, res) {
    console.log(`${req.method}`, req.url)

    //res.status(401).send({ message: 'Unauthorized' })

    if (deletedDevice) {
        loadResponseFromFile(path.join('devices', 'list', 'list-deleted-state.json'), res)
    } else {
        loadResponseFromFile(path.join('devices', 'list', 'list.json'), res)
    }
})

app.delete('/api/v1/devices', function (req, res) {
    console.log(`${req.method}`, req.url)
    deletedDevice = true
    res.send()
})

app.get('/api/v1/devices/:deviceId', deviceIdCheck, function (req, res) {
    checkError(req, res)
    loadResponseFromFile(path.join('devices', 'detail', `${req.params['deviceId']}.json`), res)
})

app.get('/api/v1/devices/:deviceId/pending-commands', deviceIdCheck, function (req, res) {
    checkError(req, res)
    res.send()
})

app.get('/api/v1/devices/:deviceId/resources', deviceIdCheck, function (req, res) {
    checkError(req, res)
    res.send()
})

app.put('/api/v1/devices/:deviceId/metadata', deviceIdCheck, function (req, res) {
    checkError(req, res)
    res.send()
})

// change device name
app.put('/api/v1/devices/:deviceId/resources/oc/con', deviceIdCheck, function (req, res) {
    checkError(req, res)
    res.send({ n: 'New Device Name' })
})

// resource detail
app.get('/api/v1/devices/:deviceId/resources/light/1', deviceIdCheck, function (req, res) {
    checkError(req, res)
    loadResponseFromFile(path.join('devices', 'detail', `${req.params['deviceId']}-resources-light-1.json`), res)
})

// resource detail
app.get('/api/v1/devices/:deviceId/resources/.well-known/wot', deviceIdCheck, function (req, res) {
    checkError(req, res)
    loadResponseFromFile(path.join('devices', 'detail', `${req.params['deviceId']}-resources-well-known-wot.json`), res)
})

app.get('/api/v1/devices/:deviceId/resources/color', deviceIdCheck, function (req, res) {
    checkError(req, res)

    if (resourceColorUpdatedValue) {
        loadResponseFromFile(path.join('devices', 'detail', `${req.params['deviceId']}-resources-color-update.json`), res)
    } else {
        loadResponseFromFile(path.join('devices', 'detail', `${req.params['deviceId']}-resources-color.json`), res)
    }
})

app.put('/api/v1/devices/:deviceId/resources/color', deviceIdCheck, function (req, res) {
    checkError(req, res)
    resourceColorUpdatedValue = true
    loadResponseFromFile(path.join('devices', 'detail', `${req.params['deviceId']}-resources-color-update.json`), res)
})

// resource detail update
app.put('/api/v1/devices/:deviceId/resources/light/1', deviceIdCheck, function (req, res) {
    checkError(req, res)
    loadResponseFromFile(path.join('devices', 'detail', `${req.params['deviceId']}-resources-light-1.json`), res)
})

// ----- GENERAL for devices -----
app.get('/api/v1/resource-links', function (req, res) {
    console.log(`${req.method}`, req.url)
    loadResponseFromFile(path.join('devices', 'detail', `${req.query['device_id_filter']}-resource-links.json`), res)
})

app.get('/api/v1/provisioning-records', function (req, res) {
    console.log(`${req.method}`, req.url)
    loadResponseFromFile(path.join('devices', 'detail', `${req.query['deviceIdFilter']}-provisioning-records.json`), res)
})

app.get('/api/v1/signing/records', function (req, res) {
    console.log(`${req.method}`, req.url)
    loadResponseFromFile(path.join('devices', 'detail', `${req.query['deviceIdFilter']}-signin-records.json`), res)
})

// ----- PENDING COMMANDS -----
app.get('/api/v1/pending-commands', function (req, res) {
    console.log(`${req.method}`, req.url)
    loadResponseFromFile('pending-commands.json', res)
})

app.get('/', () => {
    console.log(`HUB API mock server listening on port ${port}`)
    deletedDevice = false
    resourceColorUpdatedValue = false
})

app.listen(port, () => {
    console.log(`HUB API mock server listening on port ${port}`)
})
