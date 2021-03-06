import PropTypes from 'prop-types'
import BButton from 'react-bootstrap/Button'
import classNames from 'classnames'

import { buttonVariants, iconPositions } from './constants'

const { PRIMARY, SECONDARY } = buttonVariants
const { ICON_LEFT, ICON_RIGHT } = iconPositions

export const Button = ({
  children,
  onClick,
  variant,
  icon,
  iconPosition,
  loading,
  className,
  ...rest
}) => {
  const renderIcon = position => {
    if (loading) {
      if (position === ICON_LEFT) {
        return <i className="m-r-5 fas left fa-spinner" />
      }
      return null
    }
    return (
      icon &&
      position === iconPosition && (
        <i
          className={classNames(
            'fas',
            {
              [position]: true,
              'm-r-5': position === ICON_LEFT,
              'm-l-5': position === ICON_RIGHT,
            },
            icon
          )}
        />
      )
    )
  }

  const handleOnClick = (...args) => {
    if (!loading && onClick) {
      onClick(...args)
    }
  }

  return (
    <BButton
      {...rest}
      className={classNames({ loading }, className)}
      variant={variant}
      onClick={handleOnClick}
    >
      {renderIcon(ICON_LEFT)}
      {children}
      {renderIcon(ICON_RIGHT)}
    </BButton>
  )
}

Button.propTypes = {
  variant: PropTypes.oneOf([PRIMARY, SECONDARY]),
  icon: PropTypes.string,
  iconPosition: PropTypes.oneOf([ICON_LEFT, ICON_RIGHT]),
  onClick: PropTypes.func,
  loading: PropTypes.bool,
  className: PropTypes.string,
}

Button.defaultProps = {
  variant: SECONDARY,
  icon: null,
  iconPosition: ICON_LEFT,
  onClick: null,
  loading: false,
  className: null,
}
