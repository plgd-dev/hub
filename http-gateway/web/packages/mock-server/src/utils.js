const { validationResult } = require('express-validator')
const fs = require('fs-extra')

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

const loadResponseStreamFromFile = (file, res, version) => {
    const targetDirectory = `${__dirname}/data`

    const dataArray = fs.readJsonSync(`${targetDirectory}/${file}`)

    dataArray.forEach((data, key) => {
        if (version) {
            if (data.result.version === version) {
                res.write(JSON.stringify(data) + `${key === dataArray.length - 1 ? '' : '\n\n'}`)
            }
        } else {
            res.write(JSON.stringify(data) + `${key === dataArray.length - 1 ? '' : '\n\n'}`)
        }
    })

    res.send()
}

module.exports = {
    checkError,
    loadResponseFromFile,
    loadResponseStreamFromFile,
}
