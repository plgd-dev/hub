import React, { PureComponent } from 'react'
import PropTypes from 'prop-types'
import JSONEditor from 'jsoneditor'
import classNames from 'classnames'
import 'jsoneditor/dist/jsoneditor.css'

import { editorModes } from './constants'

const { CODE, VIEW } = editorModes

export class Editor extends PureComponent {
  static propTypes = {
    json: PropTypes.oneOfType([
      PropTypes.string,
      PropTypes.array,
      PropTypes.object,
    ]).isRequired,
    schema: PropTypes.oneOfType([PropTypes.array, PropTypes.object]),
    onChange: PropTypes.func,
    onError: PropTypes.func,
    editorRef: PropTypes.oneOfType([
      PropTypes.func,
      PropTypes.shape({ current: PropTypes.instanceOf(Element) }),
    ]),
    containerRef: PropTypes.oneOfType([
      PropTypes.func,
      PropTypes.shape({ current: PropTypes.instanceOf(Element) }),
    ]),
    className: PropTypes.string,
    autofocus: PropTypes.bool,
    width: PropTypes.string,
    height: PropTypes.string,
    style: PropTypes.object,
    onResize: PropTypes.func,
    mode: PropTypes.oneOf([CODE, VIEW]),
  }

  static defaultProps = {
    onChange: null,
    onError: null,
    editorRef: null,
    containerRef: null,
    className: null,
    autofocus: false,
    schema: {},
    width: '100%',
    height: '300px',
    style: {},
    onResize: () => {},
    mode: CODE,
  }

  componentDidMount() {
    setTimeout(() => {
      const { json, schema, autofocus, mode } = this.props
      const options = {
        mode,
        mainMenuBar: false,
        statusBar: false,
        onChangeText: this.onChangeText,
        onValidationError: this.onValidationError,
        schema,
      }

      this.jsoneditor = new JSONEditor(this.container, options)
      if (typeof json === 'object') {
        this.jsoneditor.set(json)
      } else if (typeof json === 'string') {
        this.jsoneditor.setText(json)
      }

      if (autofocus) {
        this.jsoneditor.focus()
      }

      if (typeof ResizeObserver === 'function' && this.container) {
        this.resizeObserver = new ResizeObserver(this.handleResize)
        this.resizeObserver.observe(this.container)
      }

      this.handleEditorRef(this.jsoneditor)
    }, 1)
  }

  componentWillUnmount() {
    if (this.jsoneditor) {
      this.handleEditorRef(null)
      this.jsoneditor.destroy()
    }

    if (typeof ResizeObserver === 'function' && this.container) {
      this.resizeObserver.unobserve(this.container)
    }
  }

  // Triggers the resize method on the editor when resizing its parent container.
  handleResize = entries => {
    const { onResize } = this.props
    const { width, height } = entries?.[0]?.contentRect || {}

    onResize(width, height, () => {
      this?.jsoneditor?.aceEditor?.resize?.()
    })
  }

  onValidationError = error => {
    const { onError } = this.props

    if (onError) {
      onError(error)
    }
  }

  onChangeText = json => {
    const { onChange } = this.props

    if (onChange) {
      onChange(json)
    }
  }

  handleContainerRef = node => {
    const { containerRef } = this.props

    if (containerRef) {
      containerRef(node)
    }

    this.container = node
  }

  handleEditorRef = editor => {
    const { editorRef } = this.props

    if (editorRef) {
      editorRef(editor)
    }
  }

  render() {
    const {
      autofocus,
      className,
      editorRef,
      containerRef,
      onChange,
      onError,
      json,
      schema,
      height,
      width,
      style,
      onResize,
      disabled,
      ...rest
    } = this.props

    return (
      <div
        {...rest}
        className={classNames(className, 'editor', {
          disabled,
          resize: !!ResizeObserver,
        })}
        ref={this.handleContainerRef}
        style={{ ...style, width, height }}
      />
    )
  }
}

export * from './constants'
