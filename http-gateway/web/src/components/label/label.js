import React, { useMemo } from 'react'
import PropTypes from 'prop-types'
import classNames from 'classnames'
import startsWith from 'lodash/startsWith'
import isEmpty from 'lodash/isEmpty'
import pickBy from 'lodash/pickBy'
import { v4 as uuidv4 } from 'uuid'

import { PropTypeRef } from '@/common/shapes'

export const Label = props => {
  const labelID = useMemo(uuidv4, [])

  const {
    children,
    title,
    className,
    id,
    onClick,
    htmlFor,
    style,
    errorMessage,
    inline,
    labelRef,
    required,
    dataClassName,
    shimmering,
    ...rest
  } = props
  const dataAttributes = pickBy(props, (_, key) => startsWith(key, 'data-'))
  const titleClassName = classNames('label-title', {
    'has-error': !isEmpty(errorMessage),
  })
  return (
    <label
      {...rest}
      ref={labelRef}
      id={id}
      htmlFor={htmlFor}
      style={style}
      className={classNames('label', { inline, shimmering }, className)}
      onClick={onClick}
    >
      <div className={classNames('label-data', dataClassName)}>
        <div {...dataAttributes} id={labelID} className={titleClassName}>
          {title}
          {required && <span className="required">{'*'}</span>}
        </div>
        {React.Children.map(children, child => {
          if (!child?.props) {
            return child
          }
          if (isEmpty(child.props['aria-labelledby'])) {
            return React.cloneElement(child, {
              'aria-labelledby': labelID,
            })
          }
          return child
        })}
      </div>
      {errorMessage && (
        <div className="label-error-message">{errorMessage}</div>
      )}
    </label>
  )
}

Label.propTypes = {
  style: PropTypes.object, // eslint-disable-line
  onClick: PropTypes.func,
  id: PropTypes.string,
  className: PropTypes.string,
  title: PropTypes.node,
  errorMessage: PropTypes.string,
  required: PropTypes.bool,
  inline: PropTypes.bool,
  labelRef: PropTypeRef,
  dataClassName: PropTypes.string,
  shimmering: PropTypes.bool,
}

Label.defaultProps = {
  style: {},
  onClick: () => {},
  id: undefined,
  className: null,
  title: null,
  errorMessage: null,
  required: false,
  inline: false,
  labelRef: () => {},
  dataClassName: null,
  shimmering: false,
}
