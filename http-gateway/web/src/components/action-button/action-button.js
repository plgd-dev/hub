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
        {items.map(item => {
          return (
            <BDropdown.Item key={item.id || item.label} onClick={item.onClick}>
              {item.icon && <i className={`fas ${item.icon}`} />}
              {item.label}
            </BDropdown.Item>
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
      onClick: PropTypes.func.isRequired,
      label: PropTypes.string.isRequired,
      id: PropTypes.string,
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
