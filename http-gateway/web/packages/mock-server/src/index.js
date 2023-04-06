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

app.get('/api/v1/pending-commands', function (req, res) {
    fs.readFile(path.join(__dirname, 'data', 'pending-commands.json'), 'utf8', function (err, data) {
        console.log(data)
        res.end(data)
    })
})

app.get('/', (req, res) => {
    res.send('Hub mock-server running!')
})

app.listen(port, () => {
    console.log(`Example app listening on port ${port}`)
})
