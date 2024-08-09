const express = require('express')
const cors = require('cors')
const axios = require('axios')
const { checkError, loadResponseStreamFromFile, loadResponseFromFile } = require('./utils')
const bodyParser = require('body-parser')
const path = require('path')

const devices = require('./routes/devices')
const dps = require('./routes/dps')
const snippetService = require('./routes/snippet-service')
const apiTokens = require('./routes/api-tokens')

const app = express()
const port = 8181
let versionData = null

require('dotenv').config()

app.use(
    cors({
        origin: '*',
        methods: ['GET', 'POST', 'DELETE', 'UPDATE', 'PUT', 'PATCH'],
    })
)

app.use(bodyParser.urlencoded({ extended: true }))

// ----- PENDING COMMANDS -----
app.get('/api/v1/pending-commands', function (req, res) {
    console.log(`${req.method}`, req.url)
    loadResponseStreamFromFile('pending-commands.json', res)
})

app.get('/', () => {
    console.log(`HUB API mock server listening on port ${port}`)
})

app.get('/.well-known/configuration', (req, res) => {
    try {
        checkError(req, res)
        axios.get('https://try.plgd.cloud/.well-known/configuration').then((r) => res.send(r.data))
    } catch (e) {
        res.status(500).send(e.toString())
    }
})

app.get('/theme/theme.json', (req, res) => {
    try {
        checkError(req, res)
        axios.get('http://localhost:3000/theme/theme.json').then((r) => res.send(r.data))
    } catch (e) {
        res.status(500).send(e.toString())
    }
})

app.get('/repos/plgd-dev/hub/releases/latest', (req, res) => {
    try {
        checkError(req, res)
        if (!versionData) {
            axios
                .get('https://api.github.com/repos/plgd-dev/hub/releases/latest', {
                    headers: {
                        Authorization: `Bearer ${process.env.REACT_APP_GITHUB_TOKEN}`,
                    },
                })
                .then((r) => {
                    versionData = r.data
                    res.send(r.data)
                })
        } else {
            res.send(versionData)
        }
    } catch (e) {
        res.status(500).send(e.toString())
    }
})

app.use(devices)
app.use(dps)
app.use('/snippet-service', snippetService)
app.use('/m2m-oauth-server', apiTokens)

app.listen(port, () => {
    console.log(`HUB API mock server listening on port ${port}`)
})
