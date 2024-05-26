const express = require('express')
const cors = require('cors')
const path = require('path')
const fs = require('fs')

const app = express()
const port = 8181

let deletedDevice = false
let resourceColorUpdatedValue = false

app.use(
    cors({
        origin: '*',
        methods: ['GET', 'POST', 'DELETE', 'UPDATE', 'PUT', 'PATCH'],
    })
)

const loadResponseFromFile = (file, res) => {
    fs.readFile(file, 'utf8', function (err, data) {
        res.send(data)
    })
}

// ----- DEVICES -----
app.get('/api/v1/devices', function (req, res) {
    console.log(`${req.method}`, req.url)

    if (deletedDevice) {
        loadResponseFromFile(path.join(__dirname, 'data', 'devices', 'list', 'list-deleted-state.json'), res)
    } else {
        loadResponseFromFile(path.join(__dirname, 'data', 'devices', 'list', 'list.json'), res)
    }
})

app.delete('/api/v1/devices', function (req, res) {
    console.log(`${req.method}`, req.url)
    deletedDevice = true
    res.send()
})

app.get('/api/v1/devices/:deviceId', function (req, res) {
    console.log(`${req.method}`, req.url)
    loadResponseFromFile(path.join(__dirname, 'data', 'devices', 'detail', `${req.params['deviceId']}.json`), res)
})

app.get('/api/v1/devices/:deviceId/pending-commands', function (req, res) {
    console.log(`${req.method}`, req.url)
    res.send()
})

app.get('/api/v1/devices/:deviceId/resources', function (req, res) {
    console.log(`${req.method}`, req.url)
    res.send()
})

app.put('/api/v1/devices/:deviceId/metadata', function (req, res) {
    console.log(`${req.method}`, req.url)
    res.send()
})

// change device name
app.put('/api/v1/devices/:deviceId/resources/oc/con', function (req, res) {
    console.log(`${req.method}`, req.url)
    res.send({ n: 'New Device Name' })
})

// resource detail
app.get('/api/v1/devices/:deviceId/resources/light/1', function (req, res) {
    console.log(`${req.method}`, req.url)
    loadResponseFromFile(path.join(__dirname, 'data', 'devices', 'detail', `${req.params['deviceId']}-resources-light-1.json`), res)
})

// resource detail
app.get('/api/v1/devices/:deviceId/resources/.well-known/wot', function (req, res) {
    console.log(`${req.method}`, req.url)
    loadResponseFromFile(path.join(__dirname, 'data', 'devices', 'detail', `${req.params['deviceId']}-resources-well-known-wot.json`), res)
})

app.get('/api/v1/devices/:deviceId/resources/color', function (req, res) {
    if (resourceColorUpdatedValue) {
        loadResponseFromFile(path.join(__dirname, 'data', 'devices', 'detail', `${req.params['deviceId']}-resources-color-update.json`), res)
    } else {
        loadResponseFromFile(path.join(__dirname, 'data', 'devices', 'detail', `${req.params['deviceId']}-resources-color.json`), res)
    }
})

app.put('/api/v1/devices/:deviceId/resources/color', function (req, res) {
    console.log(`${req.method}`, req.url)
    resourceColorUpdatedValue = true
    loadResponseFromFile(path.join(__dirname, 'data', 'devices', 'detail', `${req.params['deviceId']}-resources-color-update.json`), res)
})

// resource detail update
app.put('/api/v1/devices/:deviceId/resources/light/1', function (req, res) {
    console.log(`${req.method}`, req.url)
    loadResponseFromFile(path.join(__dirname, 'data', 'devices', 'detail', `${req.params['deviceId']}-resources-light-1.json`), res)
})

// ----- GENERAL for devices -----
app.get('/api/v1/resource-links', function (req, res) {
    console.log(`${req.method}`, req.url)
    loadResponseFromFile(path.join(__dirname, 'data', 'devices', 'detail', `${req.query['device_id_filter']}-resource-links.json`), res)
})

app.get('/api/v1/provisioning-records', function (req, res) {
    console.log(`${req.method}`, req.url)
    loadResponseFromFile(path.join(__dirname, 'data', 'devices', 'detail', `${req.query['deviceIdFilter']}-provisioning-records.json`), res)
})

app.get('/api/v1/signing/records', function (req, res) {
    console.log(`${req.method}`, req.url)
    loadResponseFromFile(path.join(__dirname, 'data', 'devices', 'detail', `${req.query['deviceIdFilter']}-signin-records.json`), res)
})

// ----- PENDING COMMANDS -----
app.get('/api/v1/pending-commands', function (req, res) {
    console.log(`${req.method}`, req.url)
    loadResponseFromFile('pending-commands.json', res)
})

app.get('/', (req, res) => {
    console.log(`HUB API mock server listening on port ${port}`)
    deletedDevice = false
    resourceColorUpdatedValue = false
})

app.listen(port, () => {
    console.log(`HUB API mock server listening on port ${port}`)
})
