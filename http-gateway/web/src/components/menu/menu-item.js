import classNames from 'classnames'
import OverlayTrigger from 'react-bootstrap/OverlayTrigger'
import Tooltip from 'react-bootstrap/Tooltip'
import { NavLink, useLocation } from 'react-router-dom'

export const MenuItem = ({
  children,
  tooltip,
  onClick,
  className,
  icon,
  to,
  ...rest
}) => {
  const location = useLocation()
  const menuItemClassName = classNames(
    'menu-item',
    className,
    to === '/' && location.pathname.includes('devices') ? 'active' : ''
  )

  const renderMenuItemContent = () => {
    return (
      <>
        <span className="icon">
          <i className={`fas ${icon}`} />
        </span>
        <span className="title">{children}</span>
      </>
    )
  }

  const renderMenuItem = () => {
    if (to) {
      return (
        <NavLink exact to={to} className={menuItemClassName} {...rest}>
          {renderMenuItemContent()}
        </NavLink>
      )
    }

    return (
      <div className={menuItemClassName} onClick={onClick} {...rest}>
        {renderMenuItemContent()}
      </div>
    )
  }

  if (tooltip) {
    return (
      <OverlayTrigger
        placement="right"
        overlay={
          <Tooltip
            id={`menu-item-tooltip-${tooltip.replace(/\s/g, '-')}`}
            className="plgd-tooltip"
          >
            {tooltip}
          </Tooltip>
        }
      >
        {renderMenuItem()}
      </OverlayTrigger>
    )
  }

  return renderMenuItem()
}
