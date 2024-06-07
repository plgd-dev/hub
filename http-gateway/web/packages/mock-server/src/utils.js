const { validationResult } = require('express-validator')

const checkError = (req, res) => {
    const errors = validationResult(req)
    if (!errors.isEmpty()) {
        return res.status(400).json({ errors: errors.array() })
    }

    console.log(`${req.method}`, req.url)
}

const loadResponseFromFile = (file, res) => {
    const targetDirectory = `${__dirname}/data`

    res.sendFile(file, { root: targetDirectory })
}

module.exports = {
    checkError,
    loadResponseFromFile,
}
