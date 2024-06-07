const express = require('express')
const { loadResponseFromFile } = require('../utils')
const path = require('path')

const router = express.Router()

router.get('/api/v1/configurations', (req, res) => {
    try {
        loadResponseFromFile(path.join('snippet-service', 'list', `configurations-list.json`), res)
    } catch (e) {
        res.status(500).send(e.toString())
    }
})

module.exports = router
