class _Emitter {
  constructor() {
    this.events = {}
  }

  // Register an event listener
  on(event, listener) {
    if (typeof this.events[event] !== 'object') {
      this.events[event] = []
    }
    this.events[event].push(listener)
    return () => this.off(event, listener)
  }

  // Unregister an event listener
  off(event, listener) {
    if (typeof this.events[event] === 'object') {
      const idx = this.events[event].indexOf(listener)
      if (idx > -1) {
        this.events[event].splice(idx, 1)
      }
    }
  }

  // Register an event listener which will be destroyed once it is called
  once(event, listener) {
    const remove = this.on(event, (...args) => {
      remove()
      listener.apply(this, args)
    })
  }

  // Emit an event
  emit(event, ...args) {
    const nameSpaces = event.indexOf('.') !== -1 ? event.split('.') : [event]
    let partialNameSpace = nameSpaces[0]

    nameSpaces.forEach(name => {
      if (partialNameSpace !== name) {
        partialNameSpace += `.${name}`
      }

      if (typeof this.events[partialNameSpace] === 'object') {
        this.events[partialNameSpace].forEach(listener => listener.apply(this, args))
      }
    })
  }
}

export const Emitter = new _Emitter()