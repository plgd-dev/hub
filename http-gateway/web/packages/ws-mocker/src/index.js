const websocket = require('ws')
const data = require('./data/devices-mocks')

const socket = new websocket.Server({ port: 12321 })

const connections = []
socket.on('connection', (ws) => {
    connections.push(ws)
})

// send a new message to every connection once per second
setTimeout(() => {
    // const date = new Date()
    for (const ws of connections) {
        data.forEach((d) => {
            ws.send(JSON.stringify(d))
        })
    }
}, 5000)
