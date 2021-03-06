import PropTypes from 'prop-types'
import BDropdown from 'react-bootstrap/Dropdown'

import { dropdownTypes } from './constants'

const { PRIMARY, SECONDARY, EMPTY } = dropdownTypes

export const ActionButton = ({ children, type, menuProps, items, ...rest }) => {
  return (
    <BDropdown className="action-button">
      <BDropdown.Toggle variant={type} {...rest}>
        <span />
        <span />
        <span />
      </BDropdown.Toggle>

      <BDropdown.Menu
        {...menuProps}
        popperConfig={{
          strategy: 'fixed',
          modifiers: [
            {
              name: 'offset',
              options: {
                offset: [-9, -15],
              },
            },
          ],
        }}
      >
        {items.filter(item => !item.hidden).map(item => {
          return (
            item.component || (
              <BDropdown.Item
                className="btn btn-secondary"
                key={item.id || item.label}
                onClick={item.onClick}
              >
                {item.icon && <i className={`fas ${item.icon}`} />}
                {item.label}
              </BDropdown.Item>
            )
          )
        })}
      </BDropdown.Menu>
    </BDropdown>
  )
}

ActionButton.propTypes = {
  children: PropTypes.node.isRequired,
  type: PropTypes.oneOf([PRIMARY, SECONDARY, EMPTY]),
  items: PropTypes.arrayOf(
    PropTypes.shape({
      onClick: PropTypes.func,
      label: PropTypes.string,
      id: PropTypes.string,
      hidden: PropTypes.bool,
      component: PropTypes.node,
    })
  ).isRequired,
  menuProps: PropTypes.shape({
    align: PropTypes.string,
    flip: PropTypes.bool,
  }),
}

ActionButton.defaultProps = {
  type: EMPTY,
  menuProps: {},
}
