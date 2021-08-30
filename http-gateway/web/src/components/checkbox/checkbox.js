import PropTypes from 'prop-types'
import classNames from 'classnames'

export const Checkbox = ({
  id,
  containerClassName,
  inputRef,
  label,
  checked,
  ...rest
}) => {
  return (
    <label className="plgd-checkbox" id={id}>
      <input {...rest} checked={checked} type="checkbox" ref={inputRef} />
      <span className="checkbox-item">
        <i
          className={classNames('fas', {
            'fa-check': checked,
            'fa-minus': !checked,
          })}
        />
      </span>
      {label && <div className="checkbox-label">{label}</div>}
    </label>
  )
}

Checkbox.propTypes = {
  id: PropTypes.string,
  containerClassName: PropTypes.string,
  inputRef: PropTypes.oneOfType([
    PropTypes.func,
    PropTypes.shape({ current: PropTypes.instanceOf(Element) }),
  ]),
  label: PropTypes.node,
  checked: PropTypes.bool.isRequired,
}

Checkbox.defaultProps = {
  id: null,
  containerClassName: null,
  inputRef: null,
  label: null,
}
