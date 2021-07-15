import { getAppMode } from '@/common/utils'
import { security } from './security'

class _WebSocketEventClient {
  constructor(api) {
    this.ws = null
    this.api = api

    // Time needed to reconnect to the Websocket after it was disconnected (cause of server or network failure).
    this.reconnectTimeMs = 2000

    // Maximum number of attempts to try reconnect the Websocket after it was disconnected.
    this.maxReconnectAttempts = 5

    // Current number of attempt to reconnect
    this.reconnectAttempts = 0

    // Event listener list key/value paris of correlationId: { createSubscription, listener }
    this.events = {}

    // Key/value pairs of correlationId: subscriptionId
    this.idsMap = {}

    // Time needed to retry the subscription, if the WebSocket is not yet connected
    this.retrySubscribeMs = 300

    // Time for delaying the event listener apply
    this.delayListenersMs = 350

    // this._connect()
  }

  _connect = async () => {
    const accessToken = await security.getAccessTokenSilently()({
      audience: security.getDefaultAudience(),
    })
    const { host, protocol } = new URL(security.getHttpGatewayAddress())
    const wsProtocol = protocol === 'https:' ? 'wss:' : 'ws:'

    const wsUrl = `${wsProtocol}//${host}${this.api}`

    // WS Instance
    this.ws = new WebSocket(wsUrl, ['Bearer', accessToken])

    this.ws.addEventListener('open', this._onOpen)
    this.ws.addEventListener('close', this._onClose)
    this.ws.addEventListener('error', this._onError)
  }

  _clear = () => {
    this.ws.removeEventListener('message', this._onMessage)
    this.ws.removeEventListener('open', this._onOpen)
    this.ws.removeEventListener('close', this._onClose)
    this.ws.removeEventListener('error', this._onError)
    this.ws = null
  }

  _onOpen = (...args) => {
    if (getAppMode() !== 'production') {
      console.info(
        `WebSocket connection was opened %c@ ${new Date().toUTCString()}`,
        'color: #255897;'
      )
    }

    // Reset the reconnect attempts
    this.reconnectAttempts = 0

    this.ws.addEventListener('message', this._onMessage)

    // re-send all WS subscriptions from the this.events list
    this._reSendSubscriptions()

    this.onOpen(...args)
  }

  _onClose = (...args) => {
    console.info(
      `WebSocket connection  was closed, reconnecting in ${Number(
        this.reconnectTimeMs / 1000
      ).toFixed(0)} seconds %c@ ${new Date().toUTCString()}`,
      'color: #255897;'
    )

    // Clear all event listeners and the WebSocket instance
    this._clear()

    if (this.reconnectAttempts < this.maxReconnectAttempts) {
      // After a close event, try to reconnect
      setTimeout(() => {
        if (!this.ws) {
          this._connect()
        }
      }, this.reconnectTimeMs)

      // Increment the reconnect attempts
      this.reconnectAttempts += 1
    } else {
      console.error(
        `Could not reconnect to the WebSocket %c@ ${new Date().toUTCString()}`,
        'color: #255897;'
      )
    }

    this.onClose(...args)
  }

  _onError = (...args) => {
    this.onError(...args)
  }

  _onMessage = (message) => {
    const { result, code } = JSON.parse(
      message.data
    )

    // If there is an error, ignore the message
    if (code !== undefined && code !== 0) {
      return
    }

    const { correlationId, subscriptionId, operationProcessed, ...args } = result || {}

    if (!correlationId) {
      return
    }

    // Save the subscriptionId to this.idsMap
    if (!this.idsMap[correlationId] && operationProcessed?.errorStatus?.code === 'OK') {
      this.idsMap[correlationId] = subscriptionId
    }

    // Invoke the event listener attached to this correlation id
    if (this.events?.[correlationId]?.listenerEnabled && !operationProcessed) {
      this.events[correlationId]?.listener(args)
    }
  }

  _reSendSubscriptions = () => {
    Object.keys(this.events).forEach(correlationId => {
      this.send({
        createSubscription: this.events[correlationId].createSubscription,
        correlationId,
      })

      // Disable the event listener since we are expecting some initial event messages
      if (this.events[correlationId]) {
        this.events[correlationId].listenerEnabled = false
      }

      // Only enable the event listener after a certain time period, to exclude some initial event messages
      setTimeout(() => {
        if (this.events[correlationId]) {
          this.events[correlationId].listenerEnabled = true
        }
      }, this.delayListenersMs)
    })
  }

  send = (data) => {
    this.ws.send(JSON.stringify(data))
  }

  subscribe = (createSubscription, correlationId, listener) => {
    if (this?.ws?.readyState !== 1) {
      setTimeout(
        () => this.subscribe(createSubscription, correlationId, listener),
        this.retrySubscribeMs
      )
      return
    }

    if (!this.events[correlationId]) {
      // Send a subscription message to the WS
      this.send({ createSubscription, correlationId })
      this.events[correlationId] = {
        createSubscription,
        listener,
        listenerEnabled: false,
      }

      // Only enable the event listener after a certain time period, to exclude some initial event messages
      setTimeout(() => {
        if (this.events[correlationId]) {
          this.events[correlationId].listenerEnabled = true
        }
      }, this.delayListenersMs)
    } else {
      console.log('already subscribed')
    }
  }

  unsubscribe = (correlationId) => {
    if (this?.ws?.readyState !== 1) {
      setTimeout(() => this.unsubscribe(correlationId), this.retrySubscribeMs)
      return
    }

    if (this.events[correlationId] && this.idsMap[correlationId]) {
      // Send an unsubscription message to the WS
      this.send({
        cancelSubscription: {
          subscriptionId: this.idsMap[correlationId],
        },
      })

      delete this.events[correlationId]
      delete this.idsMap[correlationId]
    }
  }

  onOpen = () => {}

  onClose = () => {}

  onError = () => {}
}

export const WebSocketEventClient = new _WebSocketEventClient(
  '/api/v1/ws/events'
)

/*
const ws = new WebSocketClient('wss://...')

const handler = data => update.redux()
const handler2 = data => update2.redux()

const id = ws.subscribe({
  "eventFilter": [
    "REGISTERED"
  ],
  "deviceIdFilter": [
    "string"
  ],
  "resourceIdFilter": [
    "string"
  ]
}, 'id-123456789', handler)

ws.unsubscribe('id-123456789')

--------- On Open subscriptions ----------
let initialized = false
const handler = () => {}

WebSocketEventClient.onOpen = () => {
  // On open is called even on a successfull reconnect.
  // We don't want to manually re-subscribe to a message, since this WS client is doing it automatically.
  // (When reconnected, all registered messages - Object.keys(this.events) - are sent to the websocket to recreate the subscription,
  // the handlers are untouched, still registered, no need to take action there).
  if (!initialized) {
    WebSocketEventClient.subscribe({
      "eventFilter": [
        "REGISTERED"
      ],
      "deviceIdFilter": [
        "string"
      ],
      "resourceIdFilter": [
        "string"
      ]
    }, 'id-123456789', handler)
    initialized = true
  }
}
*/
