import PropTypes from 'prop-types'
import classNames from 'classnames'

export const TreeExpander = ({ expanded, ...rest }) => {
  return (
    <div {...rest} className={classNames('tree-expander', { expanded })}>
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
