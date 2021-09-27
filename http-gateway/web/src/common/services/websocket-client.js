import { security } from './security'

export class WebSocketClient {
  constructor(api) {
    this.ws = null
    this.api = api
  }

  connect = delayMessage => {
    const { host, protocol } = new URL(security.getGeneralConfig().httpGatewayAddress)
    const wsProtocol = protocol === 'https:' ? 'wss:' : 'ws:'

    const wsUrl = `${wsProtocol}//${host}${this.api}`

    // WS Instance
    this.ws = new WebSocket(wsUrl)

    this.ws.onopen = (...args) => {
      this._sendToken()
      this._onOpen(...args)
    }

    this.ws.addEventListener('close', this._close)
    this.ws.addEventListener('error', this._error)

    // Start listening to messages after a given time.
    setTimeout(() => {
      if (this.ws) {
        this.ws.addEventListener('message', this._onMessage)
      }
    }, delayMessage)
  }

  disconnect = () => {
    if (this.ws) {
      this.ws.removeEventListener('message', this._onMessage)
      this.ws.close()
    }
  }

  close = () => {
    this.ws = null
  }

  resetListeners = () => {
    this.onOpen = () => {}
    this.onClose = () => {}
    this.onError = () => {}
    this.onMessage = () => {}
  }

  onOpen = () => {}

  onClose = () => {}

  onError = () => {}

  onMessage = () => {}

  _onOpen = (...args) => {
    this.onOpen(...args)
  }

  _close = (...args) => {
    this.close()
    this.onClose(...args)
  }

  _error = (...args) => {
    this.onError(...args)
  }

  _onMessage = (...args) => {
    this.onMessage(...args)
  }

  // Sends the access token as a first message right after connect
  _sendToken = async () => {
    const accessToken = await security.getAccessTokenSilently()({
      audience: security.getWebOAuthConfig()?.audience || '',
    })
    this.ws.send(JSON.stringify({ token: accessToken }))
  }
}
