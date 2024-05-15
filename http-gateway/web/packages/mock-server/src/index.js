const express = require('express')
const cors = require('cors')
const path = require('path')
const fs = require('fs')

const app = express()
const port = 8181

app.use(
    cors({
        origin: '*',
        methods: ['GET', 'POST', 'DELETE', 'UPDATE', 'PUT', 'PATCH'],
    })
)

const loadResponseFromFile = (filename, res) => {
    fs.readFile(path.join(__dirname, 'data', filename), 'utf8', function (err, data) {
        res.send(data)
    })
}

// ----- DEVICES -----
app.get('/api/v1/devices', function (req, res) {
    console.log('GET /api/v1/devices')
    loadResponseFromFile('devices-list.json', res)
})

app.get('/api/v1/devices/:deviceId', function (req, res) {
    console.log('GET /api/v1/devices/:deviceId')
    loadResponseFromFile(`devices-detail-${req.params['deviceId']}.json`, res)
})

app.get('/api/v1/devices/:deviceId/pending-commands', function (req, res) {
    console.log('GET /api/v1/devices/:deviceId/pending-commands')
    res.send()
})

app.get('/api/v1/devices/:deviceId/resources', function (req, res) {
    console.log('GET /api/v1/devices/:deviceId/resources')
    res.send()
})

// ----- GENERAL for devices -----

app.get('/api/v1/resource-links', function (req, res) {
    console.log(`GET /api/v1/resource-links with device_id_filter=${req.query['device_id_filter']}`)
    loadResponseFromFile(`devices-detail-${req.query['device_id_filter']}-resource-links.json`, res)
})

app.get('/api/v1/provisioning-records', function (req, res) {
    console.log(`GET /api/v1/provisioning-records with deviceIdFilter=${req.query['deviceIdFilter']}`)
    loadResponseFromFile(`devices-detail-${req.query['deviceIdFilter']}-provisioning-records.json`, res)
})

app.get('/api/v1/signing/records', function (req, res) {
    console.log(`GET /api/v1/signing-records with deviceIdFilter=${req.query['deviceIdFilter']}`)
    loadResponseFromFile(`devices-detail-${req.query['deviceIdFilter']}-signin-records.json`, res)
})

// ----- PENDING COMMANDS -----
app.get('/api/v1/pending-commands', function (req, res) {
    console.log('GET /api/v1/pending-commands')
    loadResponseFromFile('pending-commands.json', res)
})

app.get('/', (req, res) => {
    console.log(`HUB API mock server listening on port ${port}`)
})

app.listen(port, () => {
    console.log(`HUB API mock server listening on port ${port}`)
})
