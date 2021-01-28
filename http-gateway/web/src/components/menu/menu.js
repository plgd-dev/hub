import { memo } from 'react'
import classNames from 'classnames'
import { useIntl } from 'react-intl'
import PropTypes from 'prop-types'

import { MenuItem } from './menu-item'
import { messages as t } from './menu-i18n'
import './menu.scss'

export const Menu = memo(({ collapsed, toggleCollapsed }) => {
  const { formatMessage: _ } = useIntl()

  return (
    <nav id="menu">
      <MenuItem
        to="/"
        exact
        icon="fa-chart-bar"
        tooltip={collapsed && _(t.dashboard)}
      >
        {_(t.dashboard)}
      </MenuItem>
      <MenuItem to="/things" icon="fa-list" tooltip={collapsed && _(t.things)}>
        {_(t.things)}
      </MenuItem>
      <MenuItem
        to="/notifications"
        icon="fa-bell"
        tooltip={collapsed && _(t.notifications)}
      >
        {_(t.notifications)}
      </MenuItem>
      <MenuItem
        className="collapse-menu-item"
        icon={classNames({
          'fa-arrow-left': !collapsed,
          'fa-arrow-right': collapsed,
        })}
        onClick={toggleCollapsed}
      >
        {_(t.collapse)}
      </MenuItem>
    </nav>
  )
})

Menu.propTypes = {
  collapsed: PropTypes.bool.isRequired,
  toggleCollapsed: PropTypes.func.isRequired,
}
