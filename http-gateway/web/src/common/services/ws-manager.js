import { getAppMode } from '@/common/utils'
import { WebSocketClient } from './websocket-client'

class _WSManager {
  constructor() {
    // Object containing the list of websocket clients which are about to be registered (or are already registered).
    this.wsClientList = {}

    // List of registered WebsocketClients.
    this.ws = {}

    // Flag, wether the initial WS client registration was called.
    this.isInitialized = false

    // Default time for delaying the onMessage event listener (used to wait out the initial messages from the WS).
    this.defaultDelayMessageTimeMs = 2300

    // Time needed to reconnect to the Websocket after it was disconnected (cause of server or network failure).
    this.reconnectTimeMs = 2000

    // Maximum number of attempts to try reconnect the Websocket after it was disconnected.
    this.maxReconnectAttempts = 5
  }

  /**
   * Register a WS client by creating a new WebsocketClient instance.
   * @param {String} name - Name of the WS client (unique WS instance key)
   * @param {String} api - WS endpoint url
   */
  _registerWS = (name, api) => {
    if (this.ws[name]) {
      throw new Error(`WS [${name}] is already registered.`)
    }

    this.ws[name] = new WebSocketClient(api)
  }

  /**
   * Unregisters all events from the given WS and deletes its instance.
   * @param {String} name - Name of the WS client
   */
  _unregisterWS = name => {
    if (this.ws[name]) {
      this.ws[name].onMessage = () => {}
      this.ws[name].onClose = () => {}
      this.ws[name].onOpen = () => {}
      this.ws[name].onError = () => {}
      this.ws[name].disconnect()
      delete this.ws[name]
    }
  }

  /**
   * Add a new connection to the WS Clients.
   * registerWSClients will be called after that,
   * which sets all the events and callbacks needed,
   * and triggering .connect()
   * @param { name: String, api: String, listener: Function, delayMessage: integer } config
   * name - Name of the WS client (unique WS instance key) - Required
   * api - WS endpoint url - Required
   * listener - WS listener (onMessage)
   * delayMessage - Delay time for registering the listener
   */
  addWsClient = ({
    name,
    api,
    listener = null,
    delayMessage = this.defaultDelayMessageTimeMs,
    ...rest
  }) => {
    if (name && api) {
      this.wsClientList = {
        ...this.wsClientList,
        [name]: {
          api,
          listener,
          delayMessage,
          ...rest,
        },
      }

      this.registerWSClients()
    }
  }

  /**
   * Removes a WS client.
   * @param {String} name - Name of the WS client.
   */
  removeWsClient = name => {
    this._unregisterWS(name)

    if (this.wsClientList[name]) {
      delete this.wsClientList[name]
    }
  }

  /**
   * Removes all WS clients which includes the name (or part of the name) given in the partialName argument.
   * @param {String} partialName - Partial or full name of the WS client.
   */
  removeAllByPartialNameFromWsClient = partialName => {
    Object.keys(this.ws)
      .filter(key => key.includes(partialName))
      .forEach(key => this.removeWsClient(key))
  }

  /**
   * Register all WebsocketClients, which are currently present in the wsClientList.
   * Already registered clients will be ignored.
   */
  registerWSClients = () => {
    // Register clients
    Object.keys(this.wsClientList).forEach(id => {
      // Register only if not registered already.
      if (!this.ws[id]) {
        const {
          api,
          listener,
          // delayMessage = this.defaultDelayMessageTimeMs,
          onOpen = null,
          onError = null,
        } = this.wsClientList[id]

        // Reset the reconnect attempts on register
        let reconnectAttempts = 0

        // Register the WebsocketClient
        this._registerWS(id, api)

        // Connect to the WS and Register listeners
        if (this.ws[id] && listener) {
          // this.ws[id].connect(delayMessage)
          this.ws[id].onMessage = listener

          // Reconnect on close after "this.reconnectTimeMs" seconds
          this.ws[id].onClose = () => {
            if (getAppMode() !== 'production') {
              console.info(
                `ws [${id}] was closed, reconnecting in 2 seconds %c@ ${new Date().toUTCString()}`,
                'color: #255897;'
              )
            }

            if (reconnectAttempts < this.maxReconnectAttempts) {
              // After a close event, try to reconnect
              setTimeout(() => {
                if (this.ws[id]) {
                  // this.ws[id].connect(this.defaultDelayMessageTimeMs)
                }
              }, this.reconnectTimeMs)

              // Increment the reconnect attempts
              reconnectAttempts += 1
            } else {
              console.error(
                `ws [${id}] was opened %c@ ${new Date().toUTCString()}`,
                'color: #255897;'
              )
            }
          }

          // Open callback
          this.ws[id].onOpen = (...args) => {
            // Reset the reconnect attempts
            reconnectAttempts = 0

            if (onOpen) {
              onOpen(...args)
            }

            if (getAppMode() !== 'production') {
              console.info(
                `ws [${id}] was opened %c@ ${new Date().toUTCString()}`,
                'color: #255897;'
              )
            }
          }

          // Error callback
          this.ws[id].onError = (...args) => {
            if (onError) {
              onError(...args)
            }
          }
        }
      }
    })

    this.isInitialized = true
  }

  /**
   * Unregister all WeboscketClients which are present in the wsClientList.
   */
  unregisterWSClients = () => {
    const { wsClientList } = this

    // Reset listeners and unregister all WebsocketClients
    Object.keys(wsClientList).forEach(id => {
      this._unregisterWS(id)
    })
  }
}

export const WSManager = new _WSManager()
