import PropTypes from 'prop-types'
import classNames from 'classnames'

export const ThingsDetailsTitle = ({ children, className, updateData, loading, ...rest }) => {
  return (
    <h2 {...rest} className={classNames(className, 'editable-title')}>
      {children}
    </h2>
  )
}

ThingsDetailsTitle.propTypes = {
  loading: PropTypes.bool.isRequired,
  updateData: PropTypes.func.isRequired,
  className: PropTypes.string,
}

ThingsDetailsTitle.defaultProps = {
  className: null,
}