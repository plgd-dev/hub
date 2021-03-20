import PropTypes from 'prop-types'
import classNames from 'classnames'

export const TreeExpander = ({ expanded }) => {
  return (
    <div className={classNames('tree-expander', { expanded })}>
      <i
        className={classNames('fas', {
          'fa-chevron-down': expanded,
          'fa-chevron-right': !expanded,
        })}
      />
    </div>
  )
}

TreeExpander.propTypes = {
  expanded: PropTypes.bool.isRequired,
}
